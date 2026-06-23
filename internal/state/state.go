package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateHome(home string) error {
	absolute, err := filepath.Abs(home)
	if err != nil {
		return fmt.Errorf("resolve AGENTOS_HOME: %w", err)
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve user home: %w", err)
	}
	userHome, err = filepath.Abs(userHome)
	if err != nil {
		return fmt.Errorf("resolve user home: %w", err)
	}
	if strings.EqualFold(filepath.Clean(absolute), filepath.Clean(userHome)) {
		return errors.New("AGENTOS_HOME must be a dedicated subdirectory, not the user profile root")
	}
	volumeRoot := filepath.VolumeName(absolute) + string(os.PathSeparator)
	if strings.EqualFold(filepath.Clean(absolute), filepath.Clean(volumeRoot)) {
		return errors.New("AGENTOS_HOME must be a dedicated subdirectory, not a filesystem root")
	}
	return nil
}

func EnsureDir(path string) error {
	if err := ValidateHome(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return err
	}
	return secureDir(path)
}

func EnsureFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	return secureFile(path)
}

func CheckDirPrivate(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return checkDirPrivate(path)
}
