package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkDependencyFile(filePath, packageManager, directDependent, ignoredFiles string) {
	cmd := exec.Command("dep-doctor", "diagnose", "--file", filePath, "--package", packageManager, "--ignores", ignoredFiles)
	grepCmd := exec.Command("grep", "-e", "not-maintained", "-e", "archive")
	pipe, _ := cmd.StdoutPipe()
	grepCmd.Stdin = pipe
	var result bytes.Buffer
	grepCmd.Stdout = &result
	cmd.Start()
	grepCmd.Start()
	cmd.Wait()
	grepCmd.Wait()

	if result.Len() > 0 {
		processResult(filePath, directDependent, result.String())
	}
}

func processResult(filePath, directDependent, result string) {
	scanner := bufio.NewScanner(strings.NewReader(result))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		//		warnLevel := strings.Trim(parts[0], "[]")
		packageName := parts[1]
		maintenanceStatus := strings.Trim(parts[2], "():")

		dir := filepath.Dir(filePath)
		directDependentContent, err := ioutil.ReadFile(filepath.Join(dir, directDependent))
		if err != nil {
			fmt.Println("Error reading file:", err)
			continue
		}
		if strings.Contains(string(directDependentContent), "'"+packageName+"'") {
			fmt.Printf("%s/%s,%s,%s,%s\n", dir, directDependent, packageName, maintenanceStatus, parts[3])
		}
	}
}

func checkDependencies(directDependent, allDependent, packageManager string) {
	ignoredFiles, _ := ioutil.ReadFile(".dep-doctor-ignore")
	ignoredFilesStr := strings.ReplaceAll(string(ignoredFiles), "\n", " ")

	paths, _ := filepath.Glob("**/" + allDependent)
	for _, p := range paths {
		checkDependencyFile(p, packageManager, directDependent, ignoredFilesStr)
	}
}

// nolint:forbidigo
func usage() {
	fmt.Println("Usage: gh monorepo-dep-doctor --flag value (--flag value)")
	fmt.Println("example: gh monorepo-dep-doctor --flag value (--flag value) // Description")
}

func run() error {
	checkDependencies("Gemfile", "Gemfile.lock", "bundler")
	// Add more checkDependencies calls as needed
	return nil
}

func main() {

	err := run()
	if err != nil {
		log.Fatal(err) //nolint:forbidigo
	}
}
