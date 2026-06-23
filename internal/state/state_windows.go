//go:build windows

package state

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

var broadWindowsPrincipals = []string{
	"*S-1-1-0",      // Everyone
	"*S-1-5-11",     // Authenticated Users
	"*S-1-5-32-545", // BUILTIN\\Users
}

func secureDir(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("resolve current Windows user: %w", err)
	}
	if err := icacls(path, "/inheritance:r"); err != nil {
		return err
	}
	if err := icacls(path, "/grant:r", current.Username+":(OI)(CI)F"); err != nil {
		return err
	}
	_ = icacls(append([]string{path, "/remove:g"}, broadWindowsPrincipals...)...)
	return checkDirPrivate(path)
}

func secureFile(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("resolve current Windows user: %w", err)
	}
	if err := icacls(path, "/inheritance:r"); err != nil {
		return err
	}
	if err := icacls(path, "/grant:r", current.Username+":F"); err != nil {
		return err
	}
	_ = icacls(append([]string{path, "/remove:g"}, broadWindowsPrincipals...)...)
	return nil
}

func checkDirPrivate(path string) error {
	output, err := exec.Command("icacls", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("read state directory ACL: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if containsBroadPrincipal(string(output)) {
		return fmt.Errorf("state directory %q grants access to Everyone, Authenticated Users, or BUILTIN\\Users", path)
	}
	return nil
}

func containsBroadPrincipal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "everyone") ||
		strings.Contains(lower, "authenticated users") ||
		strings.Contains(lower, "builtin\\users") ||
		strings.Contains(lower, " s-1-1-0") ||
		strings.Contains(lower, " s-1-5-11") ||
		strings.Contains(lower, " s-1-5-32-545")
}

func icacls(args ...string) error {
	output, err := exec.Command("icacls", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return nil
}
