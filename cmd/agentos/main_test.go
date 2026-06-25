package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardURLKeepsCredentialOutOfQuery(t *testing.T) {
	target := dashboardURL("127.0.0.1:7467", "0123456789abcdef", "fedcba9876543210")
	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "http" || parsed.Host != "127.0.0.1:7467" || parsed.Path != "/" {
		t.Fatalf("unexpected dashboard URL: %s", target)
	}
	if parsed.RawQuery != "" {
		t.Fatalf("credential leaked into query: %s", target)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("token") != "0123456789abcdef" {
		t.Fatalf("unexpected fragment: %q", parsed.Fragment)
	}
	if values.Has("approver_token") {
		t.Fatalf("approver credential should not be preloaded into dashboard URL: %q", parsed.Fragment)
	}
	beforeFragment := strings.Split(target, "#")[0]
	if strings.Contains(beforeFragment, "0123456789abcdef") || strings.Contains(beforeFragment, "fedcba9876543210") {
		t.Fatalf("credential appeared before URL fragment: %s", target)
	}
}

func TestValidateStateHomeRejectsBroadRoots(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err = validateStateHome(userHome); err == nil {
		t.Fatal("user profile root was accepted as AGENTOS_HOME")
	}
	if err = validateStateHome(filepath.Join(userHome, ".agentos")); err != nil {
		t.Fatalf("dedicated state directory was rejected: %v", err)
	}
	volumeRoot := filepath.VolumeName(userHome) + string(os.PathSeparator)
	if err = validateStateHome(volumeRoot); err == nil {
		t.Fatal("filesystem root was accepted as AGENTOS_HOME")
	}
}
