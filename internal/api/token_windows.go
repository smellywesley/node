//go:build windows

package api

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

func secureTokenPath(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("resolve current Windows user: %w", err)
	}
	if output, commandErr := exec.Command("icacls", path, "/inheritance:r", "/grant:r",
		current.Username+":F").CombinedOutput(); commandErr != nil {
		return fmt.Errorf("secure token file ACL: %s: %w", strings.TrimSpace(string(output)), commandErr)
	}
	return nil
}
