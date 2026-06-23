package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/agentos/agentos/internal/model"
)

const Version = "v1"

func Digest(envelope model.ActionEnvelope) (string, error) {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func Evaluate(manifest model.Manifest, req model.ToolRequest) (allowed bool, approvalRequired bool, reason string) {
	if !contains(manifest.Capabilities.Tools, req.Action) {
		return false, false, fmt.Sprintf("tool %q is not declared", req.Action)
	}
	if strings.HasPrefix(req.Action, "fs.read") && !pathAllowed(req.Resource, manifest.Capabilities.FilesystemRead) {
		return false, false, "filesystem read path is outside declared capabilities"
	}
	if strings.HasPrefix(req.Action, "fs.write") && !pathAllowed(req.Resource, manifest.Capabilities.FilesystemWrite) {
		return false, false, "filesystem write path is outside declared capabilities"
	}
	if strings.HasPrefix(req.Action, "network.") && !contains(manifest.Capabilities.NetworkDestinations, req.Resource) {
		return false, false, "network destination is outside declared capabilities"
	}
	for _, rule := range manifest.ApprovalRules {
		if rule.Action == req.Action && (rule.Match == "" || strings.Contains(req.Resource, rule.Match)) {
			return true, true, "matched approval rule"
		}
	}
	return true, false, ""
}

func pathAllowed(resource string, roots []string) bool {
	if resource == "" {
		return false
	}
	cleanResource := filepath.Clean(resource)
	for _, root := range roots {
		cleanRoot := filepath.Clean(root)
		rel, err := filepath.Rel(cleanRoot, cleanResource)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
