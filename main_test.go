package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

func TestParseIgnoreFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []IgnoreRule
		wantErr bool
	}{
		{
			name: "valid format",
			content: `# Internal gems
api/schema           # Internal gem detection in api
*/itunes_receipt_decoder

# not-maintained
*/json_spec          # Used in api, test`,
			want: []IgnoreRule{
				{App: "api", Library: "schema"},
				{App: "*", Library: "itunes_receipt_decoder"},
				{App: "*", Library: "json_spec"},
			},
			wantErr: false,
		},
		{
			name:    "invalid format",
			content: "invalid",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIgnoreFile(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIgnoreFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIgnoreFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	rules := []IgnoreRule{
		{App: "api", Library: "schema"},
		{App: "*", Library: "json_spec"},
	}

	tests := []struct {
		name       string
		appName    string
		libraryName string
		want       bool
	}{
		{
			name:       "specific app and library match",
			appName:    "api",
			libraryName: "schema",
			want:       true,
		},
		{
			name:       "wildcard app match",
			appName:    "any-app",
			libraryName: "json_spec",
			want:       true,
		},
		{
			name:       "no match",
			appName:    "api",
			libraryName: "other-lib",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnore(tt.appName, tt.libraryName, rules); got != tt.want {
				t.Errorf("shouldIgnore() = %v, want %v", got, tt.want)
			}
		})
	}
}
