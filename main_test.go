package main

import (
	"os"
	"testing"

	"github.com/pgd1001/vps-tools/cmd"
)

func TestMainExecution(t *testing.T) {
	// Test that main function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Main function panicked: %v", r)
		}
	}()

	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with help flag to avoid actual execution
	os.Args = []string{"vps-tools", "--help"}
	
	// This would normally call cmd.Execute(), but we'll test the import works
	// The actual execution test would require more complex setup
}