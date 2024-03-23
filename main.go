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

func checkDependencyFile(filePath, packageManager, directDependent, ignoredFiles string) error {
	cmd := exec.Command("dep-doctor", "diagnose", "--file", filePath, "--package", packageManager, "--ignores", ignoredFiles)

	var result bytes.Buffer
	cmd.Stdout = &result

	err := cmd.Start()
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
	cmd.Wait()

	scanner := bufio.NewScanner(&result)
	for scanner.Scan() {
		line := scanner.Text()
		// grep
		if strings.Contains(line, "(not-maintained)") || strings.Contains(line, "(archive)") {
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

		// fmt.Printf("%s/%s,%s,%s,%s\n", dir, directDependent, packageName, maintenanceStatus, url)
		// Checks for files containing directly dependent libraries and standard outputs if a match is found
		if strings.Contains(string(directDependentContent), "'"+packageName+"'") {
			fmt.Printf("%s/%s,%s,%s,%s\n", dir, directDependent, packageName, maintenanceStatus, url)
		}
	}
}

func getIgnoreString() (string, error) {
	ignoredFiles, err := os.ReadFile(".gh-monorepo-dep-doctor-ignore")
	if err != nil {
		return "", fmt.Errorf("Failed to open .gh-monorepo-dep-doctor-ignore file: %w", err)
	}
	return strings.ReplaceAll(string(ignoredFiles), "\n", " "), nil
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
