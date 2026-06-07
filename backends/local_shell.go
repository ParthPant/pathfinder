package backends

import (
	"context"
	"log/slog"
	"os/exec"
	"path/filepath"
)

type LocalShellBackend struct {
	workingDir string
	env        []string
}

func NewShellBackend(workingDir string, env []string) *LocalShellBackend {
	pwd, err := filepath.Abs(workingDir)
	if err != nil {
		slog.Error("Error creating Absolute Path", "input", workingDir)
	}
	slog.Info("Setting WorkingDir", "input", workingDir, "WorkingDir", pwd)
	return &LocalShellBackend{
		pwd,
		env,
	}
}

func (s *LocalShellBackend) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	cmd := exec.Command("bash", "-c", input.Command)
	cmd.Dir = s.workingDir
	cmd.Env = s.env

	out, err := cmd.CombinedOutput()
	return ExecuteResult{
		Command:  input.Command,
		Output:   string(out),
		Error:    err,
		ExitCode: cmd.ProcessState.ExitCode(),
	}, err
}
