package runner

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	BuildCmd string
	RunCmd   string

	mu               sync.Mutex
	cancelFunc       context.CancelFunc
	activeCmd        *exec.Cmd
	lastRestartTime  time.Time
	rapidRestartCount int
}

func NewRunner(buildCmd, runCmd string) *Runner {
	return &Runner{
		BuildCmd: buildCmd,
		RunCmd:   runCmd,
	}
}

func (r *Runner) Restart() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.killActiveProcess()

	slog.Info("Starting build process...")

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	buildSuccess := r.runCommandSync(ctx, r.BuildCmd)
	if !buildSuccess {
		slog.Error("Build failed. Waiting for next file change.")
		return
	}

	slog.Info("Build successful. Starting server...")

	r.startServer(ctx, r.RunCmd)
}

func (r *Runner) killActiveProcess() {
	if r.cancelFunc != nil {
		slog.Info("Stopping existing process...")
		r.cancelFunc()
		r.cancelFunc = nil

		if r.activeCmd != nil && r.activeCmd.Process != nil {
			pid := r.activeCmd.Process.Pid
			killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))
			killCmd.Run()

			done := make(chan error, 1)
			go func() {
				done <- r.activeCmd.Wait()
			}()

			select {
			case <-time.After(2 * time.Second):
				slog.Warn("Process taking too long to stop.")
			case <-done:
			}
		}
		r.activeCmd = nil
	}
}

func parseCommand(cmdStr string) (string, []string) {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, runeValue := range cmdStr {
		if runeValue == '"' {
			inQuotes = !inQuotes
		} else if runeValue == ' ' && !inQuotes {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(runeValue)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

func (r *Runner) runCommandSync(ctx context.Context, cmdStr string) bool {
	if cmdStr == "" {
		return true
	}

	bin, args := parseCommand(cmdStr)
	if bin == "" {
		return false
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.Canceled {
			slog.Info("Build cancelled by newer file change.")
			return false
		}
		return false
	}
	return true
}

func (r *Runner) startServer(ctx context.Context, cmdStr string) {
	if cmdStr == "" {
		return
	}

	bin, args := parseCommand(cmdStr)
	if bin == "" {
		return
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.activeCmd = cmd

	err := cmd.Start()
	if err != nil {
		slog.Error("Failed to start server process", "error", err)
		return
	}

	go func() {
		err := cmd.Wait()
		if err != nil && ctx.Err() != context.Canceled {
			slog.Warn("Server process exited", "error", err)
		} else if ctx.Err() == context.Canceled {
			slog.Info("Server process stopped intentionally.")
		} else {
			slog.Info("Server process exited cleanly.")
		}
	}()
}
