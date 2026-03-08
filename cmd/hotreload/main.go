package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"hotreload/internal/runner"
	"hotreload/internal/watcher"
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

	procRunner := runner.NewRunner(buildCmd, runCmd)

	triggerChan := make(chan struct{}, 1)

	onFileChange := func() {
		select {
		case triggerChan <- struct{}{}:
			// Signal sent successfully
		default:
		}
	}

	fsWatcher, err := watcher.New(rootDir, onFileChange)
	if err != nil {
		slog.Error("Failed to initialize watcher", "error", err)
		os.Exit(1)
	}

	if err := fsWatcher.Start(); err != nil {
		slog.Error("Failed to start listening to file events", "error", err)
		os.Exit(1)
	}
	defer fsWatcher.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Triggering initial build...")
	onFileChange()


	for {
		select {
		case <-triggerChan:
			procRunner.Restart()
		case sig := <-sigChan:
			slog.Info("Received stop signal", "signal", sig)
			os.Exit(0)
		}
	}
}
