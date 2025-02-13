package main

import (
	"os"
	"testing"
)

// TestEnableFlag tests the -e flag functionality
func TestEnableFlag(t *testing.T) {
	// Set up test arguments
	os.Args = []string{
		"killswitch",
		"-w"}

	// Patch the function
	//monkey.Patch(killswitch, func() string {
	//	return "mocked"
	//})
	//defer monkey.Unpatch(RealFunction)

	// Run main
	main()

}
