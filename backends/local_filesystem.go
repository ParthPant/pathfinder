package backends

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type LocalFileSystemBackend struct {
	root string
}

func NewLocalFileSystemBackend(workingDir string) *LocalFileSystemBackend {
	pwd, err := filepath.Abs(workingDir)
	if err != nil {
		slog.Error("Error creating Absolute Path", "input", workingDir)
	}
	slog.Info("Setting WorkingDir", "input", workingDir, "WorkingDir", pwd)
	return &LocalFileSystemBackend{
		pwd,
	}
}

func (fs *LocalFileSystemBackend) Ls(ctx context.Context, input LsInput) (LsResult, error) {
	path := filepath.Join(fs.root, input.Path)
	slog.Debug("Local File System LS", "path", path)
	files, err := os.ReadDir(path)
	if err != nil {
		return LsResult{
			[]FileInfo{},
			err,
		}, err
	}

	entries := []FileInfo{}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		entries = append(entries, FileInfo{
			Name:       info.Name(),
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().String(),
		})
	}
	return LsResult{
		entries,
		nil,
	}, nil
}

func (fs *LocalFileSystemBackend) Read(ctx context.Context, input ReadInput) (ReadResult, error) {
	path := filepath.Join(fs.root, input.Path)
	slog.Debug("Local File System Read", "path", path)

	file, err := os.Open(path)
	if err != nil {
		return ReadResult{
			Error: err,
		}, err
	}
	defer file.Close()

	info, err := os.Stat(path)

	scanner := bufio.NewScanner(file)
	for i := 0; i < input.Offset && scanner.Scan(); i++ {
	}

	builder := strings.Builder{}
	for i := 0; i < input.Limit && scanner.Scan(); i++ {
		builder.Write([]byte(scanner.Text()))
	}

	return ReadResult{
		FileInfo: FileInfo{
			Name:       file.Name(),
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().String(),
		},
		FileContent: builder.String(),
		Error:       err,
	}, err
}

func (lfs *LocalFileSystemBackend) Grep(ctx context.Context, input GrepInput) (GrepResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System Grep", "path", path)

	var matches []string

	if input.Glob != nil {
		var err error = nil
		matches, err = doublestar.Glob(os.DirFS(path), *input.Glob)
		if err != nil {
			return GrepResult{Error: err}, err
		}
	} else {
		matches = []string{path}
	}

	re := regexp.MustCompile(input.Pattern)
	grepMatches := make([]GrepMatch, 0)
	for _, filepath := range matches {
		grepMatches = append(grepMatches, grep(filepath, re)...)
	}
	return GrepResult{
		grepMatches,
		nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) Glob(ctx context.Context, input GlobInput) (GlobResult, error) {
	searchBasePath := lfs.root
	if input.Path != nil {
		searchBasePath = filepath.Join(searchBasePath, *input.Path)
	}
	slog.Debug("Local File System Glob", "path", searchBasePath, "pattern", input.Pattern)

	paths, err := doublestar.Glob(os.DirFS(searchBasePath), input.Pattern)
	if err != nil {
		return GlobResult{
			Error: err,
		}, err
	}

	entries := []FileInfo{}
	for _, fileName := range paths {
		info, err := os.Stat(filepath.Join(searchBasePath, fileName))
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		entries = append(entries, FileInfo{
			Name:       info.Name(),
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().String(),
		})
	}
	return GlobResult{
		entries,
		nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) Write(ctx context.Context, input WriteInput) (WriteResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System Write", "path", path)

	if err := os.WriteFile(path, []byte(input.Content), 0644); err != nil {
		return WriteResult{
			path,
			err,
		}, err
	}
	return WriteResult{
		path,
		nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) Edit(ctx context.Context, input EditInput) (EditResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System Edit", "path", path)

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Error(err.Error())
		return EditResult{
			Error: err,
		}, err
	}

	count := strings.Count(string(content), input.OldString)
	var output string
	if input.ReplaceAll {
		output = strings.ReplaceAll(string(content), input.OldString, input.NewString)
	} else {
		output = strings.Replace(string(content), input.OldString, input.NewString, 1)
	}
	err = os.WriteFile(path, []byte(output), 0644)
	if err != nil {
		slog.Error(err.Error())
		return EditResult{
			Path:  path,
			Error: err,
		}, err
	}

	return EditResult{
		path,
		count,
		nil,
	}, nil
}

func grep(path string, pattern *regexp.Regexp) []GrepMatch {
	file, err := os.Open(path)
	if err != nil {
		slog.Error(err.Error())
		return []GrepMatch{}
	}
	defer file.Close()

	matches := make([]GrepMatch, 0)
	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if pattern.MatchString(line) {
			matches = append(matches, GrepMatch{
				path,
				i,
				line,
				nil,
			})
		}
		i++
	}
	return matches
}
