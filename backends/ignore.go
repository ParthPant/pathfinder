package backends

import (
	"log/slog"
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

type FSIgnorePolicy interface {
	ShouldIgnore(path string) (bool, error)
}

type GitIgnorePolicy struct {
	root string
	ign  *ignore.GitIgnore
}

func NewGitIgnorePolicy(basePath string) *GitIgnorePolicy {
	gitIgnorePath := filepath.Join(basePath, ".gitignore")

	gitIgnorePath, err := filepath.Abs(gitIgnorePath)
	if err != nil {
		slog.Error("Error creating Absolute Path", "input", basePath)
		return nil
	}

	if _, err := os.Stat(gitIgnorePath); err != nil {
		slog.Warn("No .gitignore file found.")
		ign := ignore.CompileIgnoreLines()
		return &GitIgnorePolicy{
			root: basePath,
			ign:  ign,
		}
	}

	ign, err := ignore.CompileIgnoreFile(gitIgnorePath)
	if err != nil {
		slog.Error("Error compiling gitignore file", "error", err)
		return nil
	}

	return &GitIgnorePolicy{
		root: basePath,
		ign:  ign,
	}
}

func (gi *GitIgnorePolicy) ShouldIgnore(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(gi.root, abs)
	if err != nil {
		return false, err
	}

	return gi.ign.MatchesPath(rel), nil
}
