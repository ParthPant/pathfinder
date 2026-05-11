package backends

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
)

type LocalShellBackend struct {
	workingDir string
	env        map[string]string
}

func NewShellBackend(workingDir string, env map[string]string) *LocalShellBackend {
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
	cmd.Env = s.getEnv()

	out, err := cmd.CombinedOutput()
	return ExecuteResult{
		Command:  input.Command,
		Output:   string(out),
		Error:    err,
		ExitCode: cmd.ProcessState.ExitCode(),
	}, err
}

func (s *LocalShellBackend) getEnv() []string {
	var env = make([]string, 0, len(s.env))
	for key, val := range s.env {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	return env
}
