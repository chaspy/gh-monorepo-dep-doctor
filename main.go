package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkDependencyFile(filePath, packageManager, directDependent, ignoredFiles string) error {
	cmd := exec.Command("dep-doctor", "diagnose", "--file", filePath, "--package", packageManager, "--ignores", ignoredFiles)
	grepCmd := exec.Command("grep", "-e", "not-maintained", "-e", "archive")

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to create stdout pipe: %w", err)
	}

	defer pipe.Close()

	grepCmd.Stdin = pipe
	var result bytes.Buffer
	grepCmd.Stdout = &result

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start dep-doctor command: %w", err)
	}

	err = grepCmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start grep command: %w", err)
	}

	// dep-doctor command returns non-zero status code when there are warning or error
	// but we can ignore it
	cmd.Wait()

	// Also grep command returns non-zero status code when there are no matching words
	grepCmd.Wait()

	if result.Len() > 0 {
		processResult(filePath, directDependent, result.String())
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

func checkDependencies(directDependent, allDependent, packageManager string) error {
	ignoredFiles, err := os.ReadFile(".dep-doctor-ignore")
	if err != nil {
		return fmt.Errorf("Failed to open .dep-doctor-ignore file: %w", err)
	}

	ignoredFilesStr := strings.ReplaceAll(string(ignoredFiles), "\n", " ")

	paths, err := filepath.Glob("**/" + allDependent)
	if err != nil {
		return fmt.Errorf("Failed to find allDependent files: %w", err)
	}
	for _, p := range paths {
		err := checkDependencyFile(p, packageManager, directDependent, ignoredFilesStr)
		if err != nil {
			return fmt.Errorf("Failed to check dependency file: %w", err)
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
