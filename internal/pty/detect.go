package pty

import (
	"errors"
	"os/exec"
)

// ErrCLINotFound is returned by DetectCLI when the named CLI is not found on PATH.
var ErrCLINotFound = errors.New("CLI not found")

// CLISpec describes a known CLI tool that AgentHub can launch.
type CLISpec struct {
	Name        string
	DisplayName string
}

// DetectedCLI is a CLISpec whose binary was found on PATH.
type DetectedCLI struct {
	Name        string
	DisplayName string
	Path        string
}

// knownCLIs is the authoritative list of AI coding CLIs that AgentHub supports.
var knownCLIs = []CLISpec{
	{Name: "claude", DisplayName: "Claude Code"},
	{Name: "codex", DisplayName: "OpenAI Codex"},
	{Name: "gemini", DisplayName: "Gemini CLI"},
	{Name: "opencode", DisplayName: "OpenCode"},
}

// DetectCLIs scans PATH for every entry in knownCLIs and returns the subset
// that is actually installed. The returned slice is always non-nil.
func DetectCLIs() []DetectedCLI {
	result := make([]DetectedCLI, 0)
	for _, spec := range knownCLIs {
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			continue
		}
		result = append(result, DetectedCLI{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
		})
	}
	return result
}

// DetectCLI looks up a single CLI by name. Returns ErrCLINotFound if the CLI
// is not in knownCLIs or its binary is not on PATH.
func DetectCLI(name string) (*DetectedCLI, error) {
	for _, spec := range knownCLIs {
		if spec.Name != name {
			continue
		}
		path, err := exec.LookPath(spec.Name)
		if err != nil {
			return nil, ErrCLINotFound
		}
		return &DetectedCLI{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Path:        path,
		}, nil
	}
	return nil, ErrCLINotFound
}
