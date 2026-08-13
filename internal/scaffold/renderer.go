package scaffold

import (
	"fmt"
	"strings"
)

func agentsFile() string {
	return `# AGENTS.md

## Working relationship

- Treat this repository's checked-in source, build settings, schemes, and tests as the source of truth.
- Make the smallest reasonable change and preserve unrelated work.
- Ask before a change would materially alter compatibility, security, data, or project structure.

## Development

- Use Swift and Apple frameworks for application and package code.
- Use Swift Testing for new tests. Use XCTest only when a specific framework integration requires it.
- Run the narrowest relevant build or test command before reporting completion.

## Git safety

- Do not discard uncommitted work.
- Get explicit approval before pushing, creating remote resources, merging, or rewriting history.
`
}

func readmeFile(request Request) string {
	return fmt.Sprintf(`# %s

This project contains a generated main application project, a local shared-code
Swift package, and an Xcode workspace that contains both.

## Generate the Xcode project

    cd "Main Application"
    script/generate-project

Open %s.xcworkspace after generation. New tests use Swift Testing.
`, request.ProjectName, request.ProjectName)
}

func workspaceFile(request Request) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Workspace version="1.0">
   <FileRef location="group:Main Application/%[1]s.xcodeproj"></FileRef>
   <FileRef location="group:Shared Code"></FileRef>
   <FileRef location="group:README.md"></FileRef>
   <FileRef location="group:AGENTS.md"></FileRef>
</Workspace>
`, request.ProjectName)
}

func projectGlobalConfig(request Request, moduleName string) string {
	return fmt.Sprintf(`#include "Universal-Common.xcconfig"

MARKETING_VERSION = 1.0.0
CURRENT_PROJECT_VERSION = 1
JWW_PROJECT_NAME = %s
JWW_PRODUCT_MODULE_NAME = %s
JWW_ROOT_BUNDLE_IDENTIFIER = com.justinwme.%s
DEVELOPMENT_TEAM =
FRAMEWORK_SEARCH_PATHS = $(inherited) $(BUILT_PRODUCTS_DIR)
`, request.ProjectName, moduleName, bundleIdentifierComponent(request.ProjectName))
}

func appTargetConfig(request Request, moduleName string, currentPlatform platform) string {
	return fmt.Sprintf(`#include "./%s/%s-Application.xcconfig"

PRODUCT_BUNDLE_IDENTIFIER = $(JWW_ROOT_BUNDLE_IDENTIFIER).app
PRODUCT_NAME = $(JWW_PROJECT_NAME)
PRODUCT_MODULE_NAME = %sApp
ASSETCATALOG_COMPILER_APPICON_NAME = AppIcon
SWIFT_ACTIVE_COMPILATION_CONDITIONS = $(inherited) APP_TARGET
`, currentPlatform.ConfigurationDir, currentPlatform.ConfigurationDir, moduleName)
}

func testTargetConfig(request Request, moduleName string, currentPlatform platform) string {
	return fmt.Sprintf(`#include "./%s/%s-Testing.xcconfig"

PRODUCT_BUNDLE_IDENTIFIER = $(JWW_ROOT_BUNDLE_IDENTIFIER).tests
PRODUCT_NAME = %s
PRODUCT_MODULE_NAME = %sTests
TEST_HOST = %s
BUNDLE_LOADER = $(TEST_HOST)
`, currentPlatform.ConfigurationDir, currentPlatform.ConfigurationDir, request.TestTargetName, moduleName, currentPlatform.TestHost)
}

func packageFile(moduleName string, platforms []platform) string {
	packagePlatforms := make([]string, 0, len(platforms))
	for _, currentPlatform := range platforms {
		packagePlatforms = append(packagePlatforms, currentPlatform.PackagePlatform)
	}
	return fmt.Sprintf(`// swift-tools-version:6.0

import PackageDescription

let package = Package(
    name: "Shared Code",
    platforms: [%s],
    products: [
        .library(name: "%[2]sShared", targets: ["%[2]sShared"])
    ],
    targets: [
        .target(name: "%[2]sShared"),
        .testTarget(name: "%[2]sSharedTests", dependencies: ["%[2]sShared"])
    ]
)
`, strings.Join(packagePlatforms, ", "), moduleName)
}

func sharedSource(moduleName, projectName string) string {
	return fmt.Sprintf(`public enum %sShared {
    public static let projectName = %q
}
`, moduleName, projectName)
}

func sharedTests(moduleName string) string {
	return fmt.Sprintf(`import Testing
@testable import %sShared

struct %sSharedTests {
    @Test
    func projectNameIsAvailable() {
        #expect(!%sShared.projectName.isEmpty)
    }
}
`, moduleName, moduleName, moduleName)
}

func appSource(moduleName string, currentPlatform platform) string {
	return fmt.Sprintf(`import SwiftUI
import %sShared

@main
struct %sApp: App {
    var body: some Scene {
        WindowGroup {
            Text(%sShared.projectName)
        }
    }
}
`, moduleName, moduleName, moduleName)
}

func appTests(moduleName string) string {
	return fmt.Sprintf(`import Testing
@testable import %sApp

struct %sAppTests {
    @Test
    func applicationModuleIsAvailable() {
        #expect(true)
    }
}
`, moduleName, moduleName)
}

func projectYAML(request Request, moduleName string, platforms []platform) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, `name: %s

options:
  minimumXcodeGenVersion: 2.46.0
  createIntermediateGroups: true
  defaultConfig: Development

configs:
  Development: debug
  Deployment: release

configFiles:
  Development: Configuration/Configurations/Development.xcconfig
  Deployment: Configuration/Configurations/Deployment.xcconfig

fileGroups:
  - Configuration
  - ../README.md
  - ../AGENTS.md

packages:
  %sShared:
    path: ../Shared Code

targets:
`, request.ProjectName, moduleName)

	for _, currentPlatform := range platforms {
		appTarget := platformTargetName(request.AppTargetName, currentPlatform)
		testTarget := platformTargetName(request.TestTargetName, currentPlatform)
		fmt.Fprintf(&builder, `  %s:
    type: application
    platform: %s
    sources: %s
    dependencies:
      - package: %sShared
        product: %sShared
    configFiles:
      Development: Configuration/%s.xcconfig
      Deployment: Configuration/%s.xcconfig
    scheme:
      testTargets:
        - %s

  %s:
    type: bundle.unit-test
    platform: %s
    sources: Tests/%s
    dependencies:
      - target: %s
    configFiles:
      Development: Configuration/%s.xcconfig
      Deployment: Configuration/%s.xcconfig

`, appTarget, currentPlatform.XcodeGenName, currentPlatform.AppDirectory, moduleName, moduleName, appTarget, appTarget, testTarget, testTarget, currentPlatform.XcodeGenName, currentPlatform.Name, appTarget, testTarget, testTarget)
	}
	return builder.String()
}
