package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Request struct {
	OutputPath     string
	ProjectName    string
	Platforms      []string
	AppTargetName  string
	TestTargetName string
}

type platform struct {
	Name             string
	XcodeGenName     string
	PackagePlatform  string
	ConfigurationDir string
	AppDirectory     string
	TestHost         string
}

var supportedPlatforms = map[string]platform{
	"ios": {
		Name: "iOS", XcodeGenName: "iOS", PackagePlatform: `.iOS("18.4")`, ConfigurationDir: "iOS", AppDirectory: "App - iOS",
		TestHost: "$(BUILT_PRODUCTS_DIR)/$(JWW_PROJECT_NAME).app/$(BUNDLE_EXECUTABLE_FOLDER_PATH)/$(JWW_PROJECT_NAME)",
	},
	"macos": {
		Name: "macOS", XcodeGenName: "macOS", PackagePlatform: `.macOS("15.4")`, ConfigurationDir: "macOS", AppDirectory: "App - macOS",
		TestHost: "$(BUILT_PRODUCTS_DIR)/$(JWW_PROJECT_NAME).app/Contents/MacOS/$(JWW_PROJECT_NAME)",
	},
}

func Create(request Request) error {
	request, moduleName, platforms, err := validateRequest(request)
	if err != nil {
		return err
	}
	if _, err := os.Stat(request.OutputPath); err == nil {
		return fmt.Errorf("output path already exists: %s", request.OutputPath)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(request.OutputPath, 0o755); err != nil {
		return err
	}
	return writeProject(request, moduleName, platforms)
}

// CreateAndGenerate creates a complete project atomically. The final directory
// is only visible after XcodeGen has successfully produced its project.
func CreateAndGenerate(request Request, output, errorOutput io.Writer) error {
	request, moduleName, platforms, err := validateRequest(request)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("xcodegen"); err != nil {
		return fmt.Errorf("xcodegen is required to create a project: %w", err)
	}
	if _, err := os.Stat(request.OutputPath); err == nil {
		return fmt.Errorf("output path already exists: %s", request.OutputPath)
	} else if !os.IsNotExist(err) {
		return err
	}

	parentDirectory := filepath.Dir(request.OutputPath)
	if err := os.MkdirAll(parentDirectory, 0o755); err != nil {
		return err
	}
	temporaryOutputPath, err := os.MkdirTemp(parentDirectory, ".xcode-starter-pack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporaryOutputPath)

	if err := writeProjectAtPath(request, moduleName, platforms, temporaryOutputPath); err != nil {
		return err
	}
	var xcodegenOutput bytes.Buffer
	if err := GenerateXcodeProject(temporaryOutputPath, &xcodegenOutput, &xcodegenOutput); err != nil {
		if _, writeErr := errorOutput.Write(xcodegenOutput.Bytes()); writeErr != nil {
			return writeErr
		}
		return err
	}
	if err := os.Remove(filepath.Join(temporaryOutputPath, "Main Application/project.yml")); err != nil {
		return fmt.Errorf("remove generation spec: %w", err)
	}
	return os.Rename(temporaryOutputPath, request.OutputPath)
}

func validateRequest(request Request) (Request, string, []platform, error) {
	request.ProjectName = strings.TrimSpace(request.ProjectName)
	request.AppTargetName = strings.TrimSpace(request.AppTargetName)
	request.TestTargetName = strings.TrimSpace(request.TestTargetName)
	if request.OutputPath == "" || request.ProjectName == "" || request.AppTargetName == "" || request.TestTargetName == "" {
		return Request{}, "", nil, fmt.Errorf("project name, app target name, unit test target name, and output path are required")
	}
	for _, value := range []string{request.ProjectName, request.AppTargetName, request.TestTargetName} {
		if !validDisplayName.MatchString(value) {
			return Request{}, "", nil, fmt.Errorf("names may contain only letters, numbers, spaces, periods, underscores, and hyphens")
		}
	}

	platforms, err := parsePlatforms(request.Platforms)
	if err != nil {
		return Request{}, "", nil, err
	}

	moduleName := swiftIdentifier(request.ProjectName)
	if moduleName == "" {
		return Request{}, "", nil, fmt.Errorf("project name must contain at least one letter or number")
	}
	return request, moduleName, platforms, nil
}

func GenerateXcodeProject(outputPath string, output, errorOutput io.Writer) error {
	mainApplicationPath := filepath.Join(outputPath, "Main Application")
	command := exec.Command("xcodegen", "--spec", "project.yml", "--project", ".")
	command.Dir = mainApplicationPath
	command.Stdout = output
	command.Stderr = errorOutput
	if err := command.Run(); err != nil {
		return fmt.Errorf("generate Xcode project: %w", err)
	}
	return nil
}

func writeProject(request Request, moduleName string, platforms []platform) error {
	return writeProjectAtPath(request, moduleName, platforms, request.OutputPath)
}

func writeProjectAtPath(request Request, moduleName string, platforms []platform, outputPath string) error {
	if err := writeBaseAssets(outputPath); err != nil {
		return err
	}
	files := map[string]string{
		"AGENTS.md": agentsFile(),
		"README.md": readmeFile(request),
		fmt.Sprintf("%s.xcworkspace/contents.xcworkspacedata", request.ProjectName): workspaceFile(request),
		"Main Application/project.yml":                                                             projectYAML(request, moduleName, platforms),
		"Main Application/Configuration/Project-Global.xcconfig":                                   projectGlobalConfig(request, moduleName),
		"Shared Code/Package.swift":                                                                packageFile(moduleName, platforms),
		fmt.Sprintf("Shared Code/Sources/%sShared/%sShared.swift", moduleName, moduleName):         sharedSource(moduleName, request.ProjectName),
		fmt.Sprintf("Shared Code/Tests/%sSharedTests/%sSharedTests.swift", moduleName, moduleName): sharedTests(moduleName),
	}

	for _, currentPlatform := range platforms {
		files[fmt.Sprintf("Main Application/Configuration/%s.xcconfig", platformTargetName(request.AppTargetName, currentPlatform))] = appTargetConfig(request, moduleName, currentPlatform)
		files[fmt.Sprintf("Main Application/Configuration/%s.xcconfig", platformTargetName(request.TestTargetName, currentPlatform))] = testTargetConfig(request, moduleName, currentPlatform)
		files[fmt.Sprintf("Main Application/%s/%sApp.swift", currentPlatform.AppDirectory, moduleName)] = appSource(moduleName, currentPlatform)
		files[fmt.Sprintf("Main Application/Tests/%s/%sTests.swift", currentPlatform.Name, moduleName)] = appTests(moduleName)
	}

	for path, contents := range files {
		if err := writeFile(filepath.Join(outputPath, path), contents, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func platformTargetName(baseName string, currentPlatform platform) string {
	return fmt.Sprintf("%s (%s)", baseName, currentPlatform.Name)
}

func parsePlatforms(values []string) ([]platform, error) {
	seen := make(map[string]bool)
	platforms := make([]platform, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		currentPlatform, ok := supportedPlatforms[key]
		if !ok {
			return nil, fmt.Errorf("unsupported platform %q; supported platforms: iOS, macOS", strings.TrimSpace(value))
		}
		if !seen[key] {
			seen[key] = true
			platforms = append(platforms, currentPlatform)
		}
	}
	if len(platforms) == 0 {
		return nil, fmt.Errorf("select at least one platform")
	}
	sort.Slice(platforms, func(left, right int) bool { return platforms[left].Name < platforms[right].Name })
	return platforms, nil
}

var nonIdentifierCharacters = regexp.MustCompile(`[^A-Za-z0-9]+`)
var validDisplayName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`)

func bundleIdentifierComponent(value string) string {
	return strings.ToLower(strings.Join(strings.FieldsFunc(value, func(character rune) bool {
		return character == ' ' || character == '.' || character == '_' || character == '-'
	}), "-"))
}

func swiftIdentifier(value string) string {
	parts := strings.FieldsFunc(nonIdentifierCharacters.ReplaceAllString(value, " "), func(character rune) bool { return character == ' ' })
	for index, part := range parts {
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	identifier := strings.Join(parts, "")
	if identifier[0] >= '0' && identifier[0] <= '9' {
		return "Project" + identifier
	}
	return identifier
}

func writeFile(path, contents string, mode os.FileMode) error {
	return writeBytes(path, []byte(contents), mode)
}

func writeBytes(path string, contents []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, contents, mode)
}
