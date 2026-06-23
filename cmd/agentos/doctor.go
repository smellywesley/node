package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/state"
)

type diagnostic struct {
	Name   string
	Status string
	Detail string
}

func doctor(cfg config, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: agentos doctor")
	}
	checks := []diagnostic{
		checkAddress(cfg.address),
		checkStateHome(cfg.home),
		checkApproverToken(),
		checkDocker(),
	}
	failed := false
	for _, check := range checks {
		fmt.Printf("%-18s %-6s %s\n", check.Name, check.Status, check.Detail)
		if check.Status == "FAIL" {
			failed = true
		}
	}
	if failed {
		return errors.New("doctor found release-blocking local configuration issues")
	}
	return nil
}

func checkAddress(address string) diagnostic {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return diagnostic{"loopback address", "FAIL", err.Error()}
	}
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		return diagnostic{"loopback address", "FAIL", "daemon must bind to loopback in v1"}
	}
	return diagnostic{"loopback address", "PASS", address}
}

func checkStateHome(home string) diagnostic {
	if err := state.ValidateHome(home); err != nil {
		return diagnostic{"state home", "FAIL", err.Error()}
	}
	if err := state.CheckDirPrivate(home); err != nil {
		return diagnostic{"state home", "FAIL", err.Error()}
	}
	if _, err := os.Stat(home); os.IsNotExist(err) {
		return diagnostic{"state home", "PASS", home + " will be created privately"}
	}
	return diagnostic{"state home", "PASS", home + " is private"}
}

func checkApproverToken() diagnostic {
	if len(strings.TrimSpace(os.Getenv("AGENTOS_APPROVER_TOKEN"))) < 32 {
		return diagnostic{"approver token", "WARN", "set AGENTOS_APPROVER_TOKEN before approve/deny"}
	}
	return diagnostic{"approver token", "PASS", "configured"}
}

func checkDocker() diagnostic {
	path, err := exec.LookPath("docker")
	if err != nil {
		return diagnostic{"docker", "WARN", "not found; dashboard works, agent runs require Docker"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return diagnostic{"docker", "WARN", strings.TrimSpace(string(output)) + "; agent runs require a running Docker engine"}
	}
	return diagnostic{"docker", "PASS", "server " + strings.TrimSpace(string(output))}
}

func validateManifestCommand(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: agentos validate <manifest.yaml>")
	}
	manifest, warnings, err := loadAndValidateManifest(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("manifest valid: %s\n", args[0])
	fmt.Printf("image: %s\n", manifest.Image)
	fmt.Printf("adapter: %s\n", manifest.Implementation.Adapter)
	fmt.Printf("mounts: %d, tools: %d, approvals: %d\n", len(manifest.Mounts), len(manifest.Capabilities.Tools), len(manifest.ApprovalRules))
	for _, warning := range warnings {
		fmt.Println("warning:", warning)
	}
	return nil
}

func loadAndValidateManifest(path string) (model.Manifest, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.Manifest{}, nil, err
	}
	var manifest model.Manifest
	if err = yaml.Unmarshal(raw, &manifest); err != nil {
		return model.Manifest{}, nil, err
	}
	manifest.ApplyDefaults()
	if err = manifest.Validate(); err != nil {
		return model.Manifest{}, nil, err
	}
	warnings, err := validateLocalManifestEnvironment(manifest)
	return manifest, warnings, err
}

func validateLocalManifestEnvironment(manifest model.Manifest) ([]string, error) {
	var warnings []string
	for _, mount := range manifest.Mounts {
		absolute, err := filepath.Abs(mount.Source)
		if err != nil {
			return warnings, fmt.Errorf("mount source %q: %w", mount.Source, err)
		}
		info, err := os.Lstat(absolute)
		if err != nil {
			return warnings, fmt.Errorf("mount source %q: %w", mount.Source, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return warnings, fmt.Errorf("mount source %q is a symbolic link", mount.Source)
		}
		if runtime.GOOS == "windows" && disallowedWindowsPath(absolute) {
			return warnings, fmt.Errorf("mount source %q uses a disallowed Windows path form", mount.Source)
		}
	}
	for _, secret := range manifest.Capabilities.Secrets {
		if _, ok := os.LookupEnv(secret); !ok {
			return warnings, fmt.Errorf("declared secret %q is not set in the local environment", secret)
		}
	}
	if warning := validateLocalImage(manifest.Image); warning != "" {
		warnings = append(warnings, warning)
	}
	return warnings, nil
}

func validateLocalImage(image string) string {
	path, err := exec.LookPath("docker")
	if err != nil {
		return "docker not found; image availability was not checked"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "image", "inspect", image)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("docker image %q is not available locally: %s", image, strings.TrimSpace(string(output)))
	}
	return ""
}

func disallowedWindowsPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasPrefix(path, `\\`) || strings.Contains(lower, `\\.\`) ||
		strings.Contains(lower, `\\?\`) || strings.Contains(filepath.Base(path), ":")
}
