package main

import (
	"os"
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
