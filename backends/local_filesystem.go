package backends

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type LocalFileSystemBackend struct {
	root         string
	ignorePolicy FSIgnorePolicy
}

func NewLocalFileSystemBackend(workingDir string) *LocalFileSystemBackend {
	pwd, err := filepath.Abs(workingDir)
	if err != nil {
		slog.Error("Error creating Absolute Path", "input", workingDir)
		return nil
	}
	slog.Info("Setting WorkingDir", "input", workingDir, "WorkingDir", pwd)
	return &LocalFileSystemBackend{
		root:         pwd,
		ignorePolicy: NewGitIgnorePolicy(workingDir),
	}
}

func (lfs *LocalFileSystemBackend) GetRoot() string {
	return lfs.root
}

func (lfs *LocalFileSystemBackend) Ls(ctx context.Context, input LsInput) (LsResult, error) {
	basePath := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System LS", "path", basePath)
	files, err := os.ReadDir(basePath)
	if err != nil {
		return LsResult{
			[]FileInfo{},
			NewFSError(fmt.Sprintf("Error reading directory %s", input.Path)),
		}, err
	}

	entries := []FileInfo{}
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(basePath, info.Name())
		if ignore, _ := lfs.ignorePolicy.ShouldIgnore(filePath); ignore {
			slog.Debug("File Ignored", "path", filePath)
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

func (lfs *LocalFileSystemBackend) Read(ctx context.Context, input ReadInput) (ReadResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Info("Local File System Read", "path", path, "input", input)

	if ignore, _ := lfs.ignorePolicy.ShouldIgnore(filepath.Join(path)); ignore {
		err := NewFSError("Reading this file is forbidden.")
		return ReadResult{
			Error: err,
		}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return ReadResult{
			Error: NewFSError(fmt.Sprintf("Error reading %s", input.Path)),
		}, err
	}
	defer file.Close()

	info, err := os.Stat(path)
	if err != nil {
		return ReadResult{
			Error: NewFSError(fmt.Sprintf("Error reading %s", input.Path)),
		}, err
	}

	scanner := bufio.NewScanner(file)
	for i := 0; i < input.Offset && scanner.Scan(); i++ {
	}

	builder := strings.Builder{}
	for i := 0; i < input.Limit && scanner.Scan(); i++ {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	return ReadResult{
		FileInfo: FileInfo{
			Name:       input.Path,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().String(),
		},
		FileContent: strings.Trim(builder.String(), "\n"),
		Error:       nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) Grep(ctx context.Context, input GrepInput) (GrepResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System Grep", "path", path)

	var matches []string

	isDir, err := isDir(path)
	if err != nil {
		return GrepResult{
			Error: NewFSError(fmt.Sprintf("Error reading %s", input.Path)),
		}, err
	}

	if isDir && input.Glob != nil {
		var err error = nil
		matches, err = doublestar.Glob(os.DirFS(path), *input.Glob)
		if err != nil {
			return GrepResult{Error: NewFSError(fmt.Sprintf("Error reading glob %s", *input.Glob))}, err
		}
		for i := 0; i < len(matches); i++ {
			matches[i] = filepath.Join(path, matches[i])
		}
	} else {
		if isDir {
			matches = recurseDir(path)
		} else {
			matches = []string{path}
		}
	}

	slog.Debug("Grep Files", "files", matches)

	if re, err := regexp.Compile(input.Pattern); err != nil {
		return GrepResult{}, err
	} else {
		grepMatches := make([]GrepMatch, 0)
		for _, matchedPath := range matches {
			if ignore, _ := lfs.ignorePolicy.ShouldIgnore(matchedPath); ignore {
				continue
			}
			grepMatches = append(grepMatches, lfs.grep(matchedPath, re)...)
		}

		return GrepResult{
			grepMatches,
			nil,
		}, nil
	}
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
			Error: NewFSError(fmt.Sprintf("Error reading glob %s", input.Pattern)),
		}, err
	}

	entries := []FileInfo{}
	for _, fileName := range paths {
		absPath := filepath.Join(searchBasePath, fileName)
		if ignore, _ := lfs.ignorePolicy.ShouldIgnore(absPath); ignore {
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		relPath, err := lfs.relPath(filepath.Join(searchBasePath, fileName))
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		entries = append(entries, FileInfo{
			Name:       relPath,
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

	if ignore, _ := lfs.ignorePolicy.ShouldIgnore(input.Path); ignore {
		err := NewFSError("Writing to this file is forbidden.")
		return WriteResult{
			Path:  input.Path,
			Error: err,
		}, err
	}

	if err := os.WriteFile(path, []byte(input.Content), 0644); err != nil {
		return WriteResult{
			input.Path,
			NewFSError(fmt.Sprintf("Error writing to file %s", input.Path)),
		}, err
	}
	return WriteResult{
		input.Path,
		nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) Edit(ctx context.Context, input EditInput) (EditResult, error) {
	path := filepath.Join(lfs.root, input.Path)
	slog.Debug("Local File System Edit", "path", path)

	if ignore, _ := lfs.ignorePolicy.ShouldIgnore(input.Path); ignore {
		err := NewFSError("Writing to this file is forbidden.")
		return EditResult{
			Path:  input.Path,
			Error: err,
		}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Error(err.Error())
		return EditResult{
			Error: NewFSError(fmt.Sprintf("Error reading file %s", input.Path)),
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
			Path:  input.Path,
			Error: NewFSError(fmt.Sprintf("Error writing to file %s", input.Path)),
		}, err
	}

	return EditResult{
		input.Path,
		count,
		nil,
	}, nil
}

func (lfs *LocalFileSystemBackend) relPath(path string) (string, error) {
	return filepath.Rel(lfs.root, path)
}

func (lfs *LocalFileSystemBackend) grep(path string, pattern *regexp.Regexp) []GrepMatch {
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
		relPath, err := lfs.relPath(path)
		if err != nil {
			continue
		}
		if pattern.MatchString(line) {
			matches = append(matches, GrepMatch{
				relPath,
				i,
				line,
				nil,
			})
		}
		i++
	}
	return matches
}

func recurseDir(path string) []string {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return []string{}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{}
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			files = append(files, recurseDir(filepath.Join(path, entry.Name()))...)
		} else {
			files = append(files, filepath.Join(path, entry.Name()))
		}
	}

	return files
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
