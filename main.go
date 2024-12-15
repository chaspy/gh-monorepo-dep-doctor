package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type IgnoreRule struct {
	App     string
	Library string
}

func parseIgnoreFile(content string) ([]IgnoreRule, error) {
	var rules []IgnoreRule
	lines := strings.Split(content, "\n")
	
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: expected 'app,library' but got '%s'", lineNum+1, line)
		}
		
		rules = append(rules, IgnoreRule{
			App:     strings.TrimSpace(parts[0]),
			Library: strings.TrimSpace(parts[1]),
		})
	}
	
	return rules, nil
}

func shouldIgnore(appName, libraryName string, rules []IgnoreRule) bool {
	for _, rule := range rules {
		appMatch := rule.App == "*" || rule.App == appName
		libraryMatch := rule.Library == "*" || rule.Library == libraryName
		if appMatch && libraryMatch {
			return true
		}
	}
	return false
}

// Check if GITHUB_TOKEN environment variable is set
func checkGitHubToken() error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	return nil
}

func checkDependencyFile(filePath, packageManager, directDependent, ignoredFiles string) error {
	// Get the app name from the file path
	appName := strings.Split(filepath.Dir(filePath), string(os.PathSeparator))[0]

	ignoreRules, err := parseIgnoreFile(ignoredFiles)
	if err != nil {
		return fmt.Errorf("Failed to parse ignore rules: %w", err)
	}

	tempFile, err := os.CreateTemp("", "Gemfile.lock.excluded")
	if err != nil {
		return fmt.Errorf("Failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}
	defer file.Close()

	gemfile_scanner := bufio.NewScanner(file)
	writing := false // false until GEM section starts
	for gemfile_scanner.Scan() {
		line := gemfile_scanner.Text()
		if line == "GEM" {
			writing = true // start after GEM section
		}
		if writing && !strings.HasSuffix(line, "!") {
			if _, err := tempFile.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("Failed to write to temp file: %w", err)
			}
		}
	}
	if err := gemfile_scanner.Err(); err != nil {
		return fmt.Errorf("Error reading file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("Failed to close temp file: %w", err)
	}

	cmd := exec.Command("dep-doctor", "diagnose", "--file", tempFile.Name(), "--package", packageManager, "--ignores", ignoredFiles)

	var result bytes.Buffer
	cmd.Stdout = &result

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start dep-doctor command: %w", err)
	}

	// dep-doctor returns 1 as exit code when an error occurs
	// https://github.com/kyoshidajp/dep-doctor/blob/15a10de03e83b0feaa98873c0cb0f42bb65a8292/cmd/root.go#L45
	// For example, 1 is returned when the rate limit of GitHub is reached or the source code url is not found.
	// GitHub Late Limit:
	//   [error] datadog-method-tracing: non-200 OK status code: 403 Forbidden body: "{\n  \"documentation_url\": \"https://docs.github.com/free-pro-team@latest/rest/overview/rate-limits-for-the-rest-api#about-secondary-rate-limits\",\n  \"message\": \"You have exceeded a secondary rate limit. Please wait a few minutes before you try again. If you reach out to GitHub Support for help, please include the request ID 9C14:3F2DD6:9E2861F:9F4864B:65FE27FF.\"\n}"
	// Source Code URL is not found:
	//   [error] web-console: source code URL is blank
	//
	// However, the latter error occurs frequently.
	// It can occur when using an internal Gem or for other reasons.
	//
	// For this reason, we do not use the exit code for handling,
	// but we do send the error line to the standard error output.
	cmd.Wait() // nolint:errcheck

	result_scanner := bufio.NewScanner(&result)
	for result_scanner.Scan() {
		line := result_scanner.Text()
		if strings.Contains(line, "(not-maintained)") || strings.Contains(line, "(archived)") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				libraryName := parts[1]
				// Check if this library should be ignored for this app
				if shouldIgnore(appName, libraryName, ignoreRules) {
					continue
				}
			}
			processResult(filePath, directDependent, line)
		} else if strings.Contains(line, "[error]") {
			fmt.Fprintf(os.Stderr, "%s diagnose includes error:%s\n", filePath, line)
		}
	}

	return nil
}

func processResult(filePath, directDependent, result string) {
	scanner := bufio.NewScanner(strings.NewReader(result))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		packageName := parts[1]
		maintenanceStatus := strings.Trim(parts[2], "():") // (archived): -> archived
		url := parts[3]

		dir := filepath.Dir(filePath)
		directDependentContent, err := os.ReadFile(filepath.Join(dir, directDependent))
		if err != nil {
			fmt.Println("Error reading file:", err)
			continue
		}

		// Checks for files containing directly dependent libraries and standard outputs if a match is found
		if strings.Contains(string(directDependentContent), "'"+packageName+"'") {
			fmt.Printf("%s/%s,%s,%s,%s\n", dir, directDependent, packageName, maintenanceStatus, url)
		}
	}
}

func getIgnoreString() (string, error) {
	const IGNORE_FILE = ".gh-monorepo-dep-doctor-ignore"
	ignoredFiles, err := os.ReadFile(IGNORE_FILE)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("Failed to open .gh-monorepo-dep-doctor-ignore file: %w", err)
	}

	rules, err := parseIgnoreFile(string(ignoredFiles))
	if err != nil {
		return "", fmt.Errorf("Failed to parse ignore file: %w", err)
	}

	var validLibraries []string
	for _, rule := range rules {
		if rule.Library != "*" {
			validLibraries = append(validLibraries, rule.Library)
		}
	}

	return strings.Join(validLibraries, " "), nil
}

func checkDependencies(directDependent, allDependent, packageManager string) error {
	ignoredFilesStr, err := getIgnoreString()
	if err != nil {
		return fmt.Errorf("Failed to get ignore string: %w", err)
	}

	paths, err := filepath.Glob("**/" + allDependent)
	if err != nil {
		return fmt.Errorf("Failed to find allDependent files: %w", err)
	}

	maxConcurrencyStr := os.Getenv("MAX_CONCURRENCY")
	maxConcurrency, err := strconv.Atoi(maxConcurrencyStr)
	if err != nil || maxConcurrency <= 0 {
		maxConcurrency = 10 // Default to 10 if not set or set to a non-positive value
	}

	errs := make(chan error, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency) // Create a semaphore to limit concurrency

	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			sem <- struct{}{} // Acquire a token
			defer wg.Done()
			defer func() { <-sem }() // Release the token
			if err := checkDependencyFile(path, packageManager, directDependent, ignoredFilesStr); err != nil {
				errs <- fmt.Errorf("Failed to check dependency file: %w", err)
			}
		}(p)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
func run() error {
	// dep-doctor needs GITHUB_TOKEN to check dependencies
	if err := checkGitHubToken(); err != nil {
		return fmt.Errorf("Failed to check GITHUB_TOKEN: %w", err)
	}

	err := checkDependencies("Gemfile", "Gemfile.lock", "bundler")
	if err != nil {
		return fmt.Errorf("Failed to check dependencies: %w", err)
	}
	// Add more checkDependencies calls as needed
	return nil
}

func main() {

	err := run()
	if err != nil {
		log.Fatal(err) //nolint:forbidigo
	}
}
