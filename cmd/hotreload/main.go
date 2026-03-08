package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

func main() {
	var (
		rootDir  string
		buildCmd string
		runCmd   string
	)

	flag.StringVar(&rootDir, "root", ".", "Directory to watch for file changes (including subfolders)")
	flag.StringVar(&buildCmd, "build", "", "Command used to build the project when a change is detected")
	flag.StringVar(&runCmd, "exec", "", "Command used to run the built server after a successful build")
	flag.Parse()

	if rootDir == "" || buildCmd == "" || runCmd == "" {
		fmt.Println("Usage: hotreload --root <project-folder> --build \"<build-command>\" --exec \"<run-command>\"")
		os.Exit(1)
	}


	if info, err := os.Stat(rootDir); err != nil || !info.IsDir() {
		slog.Error("Invalid root directory", "path", rootDir)
		os.Exit(1)
	}

	slog.Info("Starting hotreload...", "root", rootDir)
}
