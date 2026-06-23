package main

import "fmt"

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func versionCommand(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: agentos version")
	}
	fmt.Printf("agentos %s\ncommit: %s\nbuilt: %s\n", version, commit, builtAt)
	return nil
}
