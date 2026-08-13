package command

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/justin/xcode-starter-pack/internal/scaffold"
	"github.com/spf13/cobra"
)

var createAndGenerateProject = func(request scaffold.Request, output, errorOutput io.Writer) error {
	return scaffold.CreateAndGenerate(request, output, errorOutput)
}

var selectPlatforms = func(input io.Reader, output, errorOutput io.Writer) ([]string, error) {
	terminalInput, ok := input.(terminal.FileReader)
	if !ok {
		return nil, fmt.Errorf("platform selection requires an interactive terminal")
	}
	terminalOutput, ok := output.(terminal.FileWriter)
	if !ok {
		return nil, fmt.Errorf("platform selection requires an interactive terminal")
	}

	var platforms []string
	err := survey.AskOne(
		&survey.MultiSelect{
			Message: "Platforms",
			Options: []string{"iOS", "macOS"},
		},
		&platforms,
		survey.WithValidator(survey.MinItems(1)),
		survey.WithStdio(terminalInput, terminalOutput, errorOutput),
	)
	return platforms, err
}

func NewRootCommand(input io.Reader, output, errorOutput io.Writer) *cobra.Command {
	var outputPath string
	var projectName string
	var platforms []string
	var appTargetName string
	var testTargetName string

	rootCommand := &cobra.Command{
		Use:   "xcode-starter-pack",
		Short: "Create an Xcode project from this starter pack",
	}

	newCommand := &cobra.Command{
		Use:   "new",
		Short: "Interactively create a project",
		RunE: func(command *cobra.Command, arguments []string) error {
			request, err := promptForProject(input, output, errorOutput, scaffold.Request{
				OutputPath: outputPath, ProjectName: projectName, Platforms: platforms,
				AppTargetName: appTargetName, TestTargetName: testTargetName,
			})
			if err != nil {
				return err
			}

			if err := createAndGenerateProject(request, output, errorOutput); err != nil {
				return err
			}

			_, err = fmt.Fprintf(output, "Created %s\n", request.OutputPath)
			return err
		},
	}
	newCommand.Flags().StringVarP(&outputPath, "output", "o", "", "directory for the new project")
	newCommand.Flags().StringVar(&projectName, "name", "", "project name")
	newCommand.Flags().StringSliceVarP(&platforms, "platform", "p", nil, "platform to include; repeat for multiple platforms")
	newCommand.Flags().StringVar(&appTargetName, "app-target", "", "app target base name")
	newCommand.Flags().StringVar(&testTargetName, "test-target", "", "unit test target base name")
	rootCommand.AddCommand(newCommand)
	rootCommand.SetOut(output)
	rootCommand.SetErr(errorOutput)
	return rootCommand
}

func promptForProject(input io.Reader, output, errorOutput io.Writer, request scaffold.Request) (scaffold.Request, error) {
	reader := bufio.NewReader(input)
	if request.OutputPath == "" {
		var err error
		request.OutputPath, err = prompt(reader, output, "Project output directory")
		if err != nil {
			return scaffold.Request{}, err
		}
	}
	if request.ProjectName == "" {
		var err error
		request.ProjectName, err = prompt(reader, output, "Project name")
		if err != nil {
			return scaffold.Request{}, err
		}
	}
	if len(request.Platforms) == 0 {
		var err error
		request.Platforms, err = selectPlatforms(input, output, errorOutput)
		if err != nil {
			return scaffold.Request{}, err
		}
	}
	if request.AppTargetName == "" {
		var err error
		request.AppTargetName, err = promptWithDefault(reader, output, "App target name", request.ProjectName)
		if err != nil {
			return scaffold.Request{}, err
		}
	}
	if request.TestTargetName == "" {
		var err error
		request.TestTargetName, err = promptWithDefault(reader, output, "Unit test target name", request.ProjectName+" Tests")
		if err != nil {
			return scaffold.Request{}, err
		}
	}
	return request, nil
}

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	if _, err := fmt.Fprintf(output, "%s: ", label); err != nil {
		return "", err
	}

	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptWithDefault(reader *bufio.Reader, output io.Writer, label, defaultValue string) (string, error) {
	if _, err := fmt.Fprintf(output, "%s [%s]: ", label, defaultValue); err != nil {
		return "", err
	}

	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}
