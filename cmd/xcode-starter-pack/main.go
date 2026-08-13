package main

import (
	"os"

	"github.com/justin/xcode-starter-pack/internal/command"
)

func main() {
	rootCommand := command.NewRootCommand(os.Stdin, os.Stdout, os.Stderr)
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
