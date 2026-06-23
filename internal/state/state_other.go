//go:build !windows

package state

import (
	"fmt"
	"os"
)

func secureDir(path string) error { return nil }

func secureFile(path string) error { return nil }

func checkDirPrivate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("state directory %q is accessible by group or others", path)
	}
	return nil
}
