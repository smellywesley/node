//go:build !windows

package api

import "os"

func secureTokenPath(path string) error {
	return os.Chmod(path, 0o600)
}
