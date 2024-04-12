package dcd

import (
	"testing"
)

func TestGetRepoName(t *testing.T) {
	// Define test cases
	testCases := []struct {
		gitURL       string // Input Git URL
		expectedName string // Expected repository name
		expectError  bool   // Whether an error is expected
	}{
		{"git@github.com:progsoftware/dcd.git", "dcd", false},
		{"https://github.com/progsoftware/dcd.git", "dcd", false},
		{"git@bitbucket.org:progsoftware/dcd.git", "dcd", false},
		{"https://bitbucket.org/progsoftware/dcd.git", "dcd", false},
		{"git@github.com:progsoftware/multi-part-name.git", "multi-part-name", false},
		{"https://github.com/progsoftware/with-dash.git", "with-dash", false},
		{"git@github.com:progsoftware.git", "", true}, // Invalid URL, expected to fail
		{"", "", true}, // Empty URL, expected to fail
	}

	for _, tc := range testCases {
		// Run getRepoName with the gitURL from the test case
		repoName, err := getRepoNameFromGitURL(tc.gitURL)

		// Check if an error was expected
		if tc.expectError {
			if err == nil {
				t.Errorf("Expected an error for URL '%s', but got none", tc.gitURL)
			}
			// Skip further checks if an error was expected
			continue
		} else if err != nil {
			t.Errorf("Unexpected error for URL '%s': %v", tc.gitURL, err)
			continue
		}

		// Check if the extracted repo name matches the expected name
		if repoName != tc.expectedName {
			t.Errorf("For URL '%s', expected repo name '%s', got '%s'", tc.gitURL, tc.expectedName, repoName)
		}
	}
}
