package scaffold

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndGeneratePublishesFinishedWorkspace(t *testing.T) {
	if _, err := exec.LookPath("xcodegen"); err != nil {
		t.Skip("xcodegen is required for generation integration coverage")
	}
	outputPath := filepath.Join(t.TempDir(), "Orbit Project")
	var output bytes.Buffer
	err := CreateAndGenerate(Request{
		OutputPath: outputPath, ProjectName: "Orbit Project",
		Platforms:     []string{"iOS", "macOS"},
		AppTargetName: "Orbit", TestTargetName: "Orbit Tests",
	}, &output, &output)
	if err != nil {
		t.Fatalf("CreateAndGenerate() error = %v\n%s", err, &output)
	}
	for _, relativePath := range []string{"Main Application/project.yml", "Main Application/script"} {
		if _, err := os.Stat(filepath.Join(outputPath, relativePath)); !os.IsNotExist(err) {
			t.Errorf("generation input %s exists or could not be checked: %v", relativePath, err)
		}
	}
	read := func(relativePath string) string {
		t.Helper()
		contents, err := os.ReadFile(filepath.Join(outputPath, relativePath))
		if err != nil {
			t.Fatal(err)
		}
		return string(contents)
	}
	projectPath := "Main Application/Orbit Project.xcodeproj/project.pbxproj"
	project := read(projectPath)
	for _, unwanted := range []string{"XCLocalSwiftPackageReference", "/* Packages */", "Shared Code", "README.md", "AGENTS.md", ".xcode-starter-pack-"} {
		if strings.Contains(project, unwanted) {
			t.Errorf("project contains duplicate workspace reference %q", unwanted)
		}
	}
	if count := strings.Count(project, "productName = OrbitProjectShared;"); count != 2 {
		t.Errorf("shared product dependency count = %d, want one for each app target", count)
	}
	if count := strings.Count(project, "isa = PBXBuildFile; productRef ="); count != 6 {
		t.Errorf("linked package product count = %d, want shared and toolkit products for both platforms", count)
	}
	workspace := read("Orbit Project.xcworkspace/contents.xcworkspacedata")
	for _, reference := range []string{"Main Application/Orbit Project.xcodeproj", "Shared Code", "README.md", "AGENTS.md"} {
		if strings.Count(workspace, "group:"+reference+"\"") != 1 {
			t.Errorf("workspace must contain exactly one reference to %s", reference)
		}
	}
	if _, err := exec.Command("git", "-C", outputPath, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "-C", outputPath, "-c", "core.excludesFile=/dev/null", "check-ignore", projectPath)
	if result, err := command.CombinedOutput(); err == nil {
		t.Fatalf("generated project is ignored: %s", result)
	} else if exitError, ok := err.(*exec.ExitError); !ok || exitError.ExitCode() != 1 {
		t.Fatalf("git check-ignore failed: %v\n%s", err, result)
	}
}

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
	for _, expected := range []string{
		"jww-swift-toolkit:",
		"url: https://github.com/justin/jww-swift-toolkit.git",
		"from: 1.1.0",
		"path: ../Shared Code",
		"Orbit (iOS):",
		"Orbit (macOS):",
		"Orbit Tests (iOS):",
		"Orbit Tests (macOS):",
		"package: jww-swift-toolkit\n        product: JWWCore",
		"package: jww-swift-toolkit\n        product: JWWTestExtensions",
	} {
		if !strings.Contains(string(projectYAML), expected) {
			t.Errorf("project.yml does not contain %q", expected)
		}
	}

	appSource, err := os.ReadFile(filepath.Join(outputPath, "Main Application/App - iOS/OrbitApp.swift"))
	if err != nil {
		t.Fatalf("ReadFile(app source) error = %v", err)
	}
	if !strings.Contains(string(appSource), "import JWWCore") {
		t.Error("app source does not import JWWCore")
	}

	testSource, err := os.ReadFile(filepath.Join(outputPath, "Main Application/Tests/iOS/OrbitTests.swift"))
	if err != nil {
		t.Fatalf("ReadFile(test source) error = %v", err)
	}
	if !strings.Contains(string(testSource), "import JWWTestExtensions") {
		t.Error("test source does not import JWWTestExtensions")
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
