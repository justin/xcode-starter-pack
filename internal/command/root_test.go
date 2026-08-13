package command

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justin/xcode-starter-pack/internal/scaffold"
)

func TestNewCommandPromptsAndCreatesProject(t *testing.T) {
	originalCreateAndGenerateProject := createAndGenerateProject
	t.Cleanup(func() { createAndGenerateProject = originalCreateAndGenerateProject })
	createAndGenerateProject = func(request scaffold.Request, output, errorOutput io.Writer) error {
		return scaffold.Create(request)
	}
	originalSelectPlatforms := selectPlatforms
	t.Cleanup(func() { selectPlatforms = originalSelectPlatforms })
	selectPlatforms = func(input io.Reader, output, errorOutput io.Writer) ([]string, error) { return []string{"iOS"}, nil }

	outputPath := filepath.Join(t.TempDir(), "Orbit")
	input := strings.NewReader("Orbit\nOrbit\nOrbit Tests\n")
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	rootCommand := NewRootCommand(input, &output, &errorOutput)
	rootCommand.SetArgs([]string{"new", "--output", outputPath})
	if err := rootCommand.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputPath, "AGENTS.md")); err != nil {
		t.Fatalf("expected generated AGENTS.md: %v", err)
	}
}

func TestNewCommandPromptsForOutputPathWhenFlagIsOmitted(t *testing.T) {
	originalCreateAndGenerateProject := createAndGenerateProject
	t.Cleanup(func() { createAndGenerateProject = originalCreateAndGenerateProject })
	createAndGenerateProject = func(request scaffold.Request, output, errorOutput io.Writer) error {
		return scaffold.Create(request)
	}
	originalSelectPlatforms := selectPlatforms
	t.Cleanup(func() { selectPlatforms = originalSelectPlatforms })
	selectPlatforms = func(input io.Reader, output, errorOutput io.Writer) ([]string, error) { return []string{"iOS"}, nil }

	outputPath := filepath.Join(t.TempDir(), "Orbit")
	input := strings.NewReader(outputPath + "\nOrbit\nOrbit\nOrbit Tests\n")
	var output bytes.Buffer
	var errorOutput bytes.Buffer

	rootCommand := NewRootCommand(input, &output, &errorOutput)
	rootCommand.SetArgs([]string{"new"})
	if err := rootCommand.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputPath, "AGENTS.md")); err != nil {
		t.Fatalf("expected generated AGENTS.md: %v", err)
	}
	if !strings.HasPrefix(output.String(), "Project output directory: ") {
		t.Errorf("prompt output = %q, want output directory prompt first", output.String())
	}
}

func TestNewCommandAcceptsFullyNonInteractiveFlags(t *testing.T) {
	originalCreateAndGenerateProject := createAndGenerateProject
	t.Cleanup(func() { createAndGenerateProject = originalCreateAndGenerateProject })
	createAndGenerateProject = func(request scaffold.Request, output, errorOutput io.Writer) error {
		return scaffold.Create(request)
	}

	outputPath := filepath.Join(t.TempDir(), "Orbit")
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	rootCommand := NewRootCommand(strings.NewReader(""), &output, &errorOutput)
	rootCommand.SetArgs([]string{
		"new",
		"--output", outputPath,
		"--name", "Orbit",
		"--platform", "iOS",
		"--app-target", "Orbit",
		"--test-target", "Orbit Tests",
	})
	if err := rootCommand.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(output.String(), "Project name:") {
		t.Errorf("output unexpectedly contains a prompt: %q", output.String())
	}
}
