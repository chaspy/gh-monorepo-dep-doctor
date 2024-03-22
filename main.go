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

	grepCmd.Stdin = pipe
	var result bytes.Buffer
	grepCmd.Stdout = &result

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start cmd: %w", err)
	}

	err = grepCmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start grepCmd: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Failed to wait for cmd: %w", err)
	}

	err = grepCmd.Wait()
	if err != nil {
		return fmt.Errorf("Failed to wait for grepCmd: %w", err)
	}

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
		maintenanceStatus := strings.Trim(parts[2], "():")

		dir := filepath.Dir(filePath)
		directDependentContent, err := os.ReadFile(filepath.Join(dir, directDependent))
		if err != nil {
			fmt.Println("Error reading file:", err)
			continue
		}
		if strings.Contains(string(directDependentContent), "'"+packageName+"'") {
			fmt.Printf("%s/%s,%s,%s,%s\n", dir, directDependent, packageName, maintenanceStatus, parts[3])
		}
	}
}

func checkDependencies(directDependent, allDependent, packageManager string) error {
	ignoredFiles, err := os.ReadFile(".dep-doctor-ignore")
	if err != nil {
		return fmt.Errorf("Failed to open .dep-doctor-ignore file: %w", err)
	}

	ignoredFilesStr := strings.ReplaceAll(string(ignoredFiles), "\n", " ")

	paths, _ := filepath.Glob("**/" + allDependent)
	for _, p := range paths {
		err := checkDependencyFile(p, packageManager, directDependent, ignoredFilesStr)
		if err != nil {
			return fmt.Errorf("Failed to check dependency file: %w", err)
		}
	}
	return nil
}

// nolint:forbidigo
func usage() {
	fmt.Println("Usage: gh monorepo-dep-doctor --flag value (--flag value)")
	fmt.Println("example: gh monorepo-dep-doctor --flag value (--flag value) // Description")
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
