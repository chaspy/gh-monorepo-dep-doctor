package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestGetIgnoreString tests the getIgnoreString function for correct behavior.
func TestGetIgnoreString(t *testing.T) {
	// Setup: Create a temporary ignore file.
	const testIgnoreContent = `# This is a comment
ignored_lib1 # This is a comment

ignored_lib2
# Another comment
`
	const ignoreFileName = ".gh-monorepo-dep-doctor-ignore"
	err := os.WriteFile(ignoreFileName, []byte(testIgnoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary ignore file: %v", err)
	}
	defer os.Remove(ignoreFileName) // Cleanup after the test

	// Execute the function under test.
	ignoreString, err := getIgnoreString()
	if err != nil {
		t.Fatalf("getIgnoreString returned an error: %v", err)
	}

	// Verify the result.
	expected := "ignored_lib1 ignored_lib2"
	if ignoreString != expected {
		t.Errorf("Expected ignore string to be '%s', got '%s'", expected, ignoreString)
	}
}

// TestProcessResult tests the processResult function.
func TestProcessResult(t *testing.T) {
	// Setup: Create a temporary directory and a dummy direct dependent file.
	tempDir, err := os.MkdirTemp("", "testdeps")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup after the test

	dummyDirectDependentFile := filepath.Join(tempDir, "Gemfile")
	dummyContent := "gem 'dummy_package'\ngem 'another_package'"
	if err := os.WriteFile(dummyDirectDependentFile, []byte(dummyContent), 0644); err != nil {
		t.Fatalf("Failed to create dummy direct dependent file: %v", err)
	}

	// Define a test case with expected output.
	testCases := []struct {
		filePath        string
		directDependent string
		result          string
		expectedOutput  string
	}{
		{
			filePath:        dummyDirectDependentFile,
			directDependent: "Gemfile",
			result:          "[warning] dummy_package (not-maintained): https://example.com/dummy_package",
			expectedOutput:  tempDir + "/Gemfile,dummy_package,not-maintained,https://example.com/dummy_package\n",
		},
		{
			filePath:        dummyDirectDependentFile,
			directDependent: "Gemfile",
			result:          "[warning] another_package (archived): https://example.com/another_package",
			expectedOutput:  tempDir + "/Gemfile,another_package,archived,https://example.com/another_package\n",
		},
	}

	for _, testcase := range testCases {
		// Capture the output.
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Execute the function under test.
		processResult(testcase.filePath, testcase.directDependent, testcase.result)

		// Read and restore the output.
		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = oldStdout

		// Verify the result.
		if gotOutput := string(out); gotOutput != testcase.expectedOutput {
			t.Errorf("Expected output to be '%s', got '%s'", testcase.expectedOutput, gotOutput)
		}
	}
}
