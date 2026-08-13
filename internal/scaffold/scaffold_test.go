package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateBuildsReactionsStyleProjectLayout(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "Orbit")

	err := Create(Request{
		OutputPath:     outputPath,
		ProjectName:    "Orbit",
		Platforms:      []string{"iOS", "macOS"},
		AppTargetName:  "Orbit",
		TestTargetName: "Orbit Tests",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for _, relativePath := range []string{
		"AGENTS.md",
		".gitignore",
		".swiftlint.yml",
		"Orbit.xcworkspace/contents.xcworkspacedata",
		"Main Application/project.yml",
		"Main Application/Configuration/Orbit (iOS).xcconfig",
		"Main Application/Configuration/Orbit (macOS).xcconfig",
		"Main Application/Tests/iOS/OrbitTests.swift",
		"Shared Code/Package.swift",
		"Shared Code/Sources/OrbitShared/OrbitShared.swift",
	} {
		if _, err := os.Stat(filepath.Join(outputPath, relativePath)); err != nil {
			t.Errorf("expected %s to exist: %v", relativePath, err)
		}
	}

	projectYAML, err := os.ReadFile(filepath.Join(outputPath, "Main Application/project.yml"))
	if err != nil {
		t.Fatalf("ReadFile(project.yml) error = %v", err)
	}

	projectGlobalConfig, err := os.ReadFile(filepath.Join(outputPath, "Main Application/Configuration/Project-Global.xcconfig"))
	if err != nil {
		t.Fatalf("ReadFile(Project-Global.xcconfig) error = %v", err)
	}
	if !strings.Contains(string(projectGlobalConfig), "JWW_ROOT_BUNDLE_IDENTIFIER = com.justinwme.orbit") {
		t.Errorf("Project-Global.xcconfig does not contain the expected bundle identifier root")
	}

	swiftLintConfiguration, err := os.ReadFile(filepath.Join(outputPath, ".swiftlint.yml"))
	if err != nil {
		t.Fatalf("ReadFile(.swiftlint.yml) error = %v", err)
	}
	if !strings.Contains(string(swiftLintConfiguration), "  - Main Application") {
		t.Error(".swiftlint.yml does not include generated application sources")
	}

	for _, relativePath := range []string{
		"Main Application/Configuration/Orbit (iOS).xcconfig",
		"Main Application/Configuration/Orbit Tests (iOS).xcconfig",
	} {
		contents, err := os.ReadFile(filepath.Join(outputPath, relativePath))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", relativePath, err)
		}
		if relativePath == "Main Application/Configuration/Orbit (iOS).xcconfig" && !strings.Contains(string(contents), "PRODUCT_BUNDLE_IDENTIFIER = $(JWW_ROOT_BUNDLE_IDENTIFIER).app") {
			t.Errorf("%s does not contain the app bundle identifier", relativePath)
		}
		if relativePath == "Main Application/Configuration/Orbit Tests (iOS).xcconfig" && !strings.Contains(string(contents), "PRODUCT_BUNDLE_IDENTIFIER = $(JWW_ROOT_BUNDLE_IDENTIFIER).tests") {
			t.Errorf("%s does not contain the test bundle identifier", relativePath)
		}
	}
	for _, expected := range []string{"path: ../Shared Code", "Orbit (iOS):", "Orbit (macOS):", "Orbit Tests (iOS):", "Orbit Tests (macOS):"} {
		if !strings.Contains(string(projectYAML), expected) {
			t.Errorf("project.yml does not contain %q", expected)
		}
	}
	if strings.Contains(string(projectYAML), "Profiling") {
		t.Error("project.yml includes the removed Profiling configuration")
	}
	if _, err := os.Stat(filepath.Join(outputPath, "Main Application/Configuration/Configurations/Profiling.xcconfig")); !os.IsNotExist(err) {
		t.Errorf("Profiling.xcconfig exists or could not be checked: %v", err)
	}
}

func TestCreateRejectsUnsupportedPlatform(t *testing.T) {
	err := Create(Request{
		OutputPath:     filepath.Join(t.TempDir(), "Orbit"),
		ProjectName:    "Orbit",
		Platforms:      []string{"visionOS"},
		AppTargetName:  "Orbit",
		TestTargetName: "Orbit Tests",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("Create() error = %v, want unsupported platform error", err)
	}
}

func TestCreateMakesANumberedProjectNameAValidSwiftIdentifier(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "2026 Orbit")
	err := Create(Request{
		OutputPath:     outputPath,
		ProjectName:    "2026 Orbit",
		Platforms:      []string{"iOS"},
		AppTargetName:  "Orbit",
		TestTargetName: "Orbit Tests",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sharedSource, err := os.ReadFile(filepath.Join(outputPath, "Shared Code/Sources/Project2026OrbitShared/Project2026OrbitShared.swift"))
	if err != nil {
		t.Fatalf("ReadFile(shared source) error = %v", err)
	}
	if !strings.Contains(string(sharedSource), "enum Project2026OrbitShared") {
		t.Errorf("shared source = %q, want valid Swift identifier", sharedSource)
	}
}

func TestBundleIdentifierComponentUsesHyphenSeparatedLowercaseWords(t *testing.T) {
	if got, want := bundleIdentifierComponent("My_Project.Name"), "my-project-name"; got != want {
		t.Errorf("bundleIdentifierComponent() = %q, want %q", got, want)
	}
}

func TestCreatePreservesSpacesInTargetNames(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "Test Project 1")
	err := Create(Request{
		OutputPath:     outputPath,
		ProjectName:    "Test Project 1",
		Platforms:      []string{"iOS", "macOS"},
		AppTargetName:  "Test Project 1",
		TestTargetName: "Test Project 1 Tests",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	projectYAML, err := os.ReadFile(filepath.Join(outputPath, "Main Application/project.yml"))
	if err != nil {
		t.Fatalf("ReadFile(project.yml) error = %v", err)
	}
	for _, expected := range []string{
		"Test Project 1 (iOS):",
		"Test Project 1 (macOS):",
		"Test Project 1 Tests (iOS):",
		"Test Project 1 Tests (macOS):",
	} {
		if !strings.Contains(string(projectYAML), expected) {
			t.Errorf("project.yml does not contain %q", expected)
		}
	}
}
