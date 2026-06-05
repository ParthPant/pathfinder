package backends

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{
		"*.go",
	}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)
	os.WriteFile(filepath.Join(dir, "test.go"), []byte{}, 0644)

	os.WriteFile(filepath.Join(dir, "one.txt"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "two.txt"), []byte{}, 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(filepath.Join(dir, "subdir", "three.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)

	t.Logf("Working Dir: %s", fs.root)

	ctx := context.Background()
	path := "/"
	input := LsInput{
		Path: path,
	}

	res, err := fs.Ls(ctx, input)
	if err != nil {
		t.Error(err)
	}

	t.Logf("Ls Result %v", res.Entries)

	expected := []string{"one.txt", "two.txt", "subdir", ".gitignore"}
	slices.Sort(expected)

	for i, entry := range res.Entries {
		if entry.Name != expected[i] {
			t.Errorf("Found %s, want %s", entry.Name, expected[i])
		}
	}
}

func TestDoubleStarGlob(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "one.txt"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "two.txt"), []byte{}, 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(filepath.Join(dir, "subdir", "three.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)

	t.Logf("Working Dir: %s", fs.root)

	ctx := context.Background()
	path := "/"
	input := GlobInput{
		Pattern: "**/*.txt",
		Path:    &path,
	}
	out, err := fs.Glob(ctx, input)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%d Matches Found, Error: %v", len(out.Matches), out.Error)
	expected := []string{"one.txt", "two.txt", "subdir/three.txt"}
	for i, match := range out.Matches {
		if match.Name != expected[i] {
			t.Errorf("Found %s, want %s", match.Name, expected[i])
		}
	}

	if len(out.Matches) != 3 {
		t.Errorf("Found %d, Expected %d", len(out.Matches), 3)
	}
}

func TestRead(t *testing.T) {
	dir := t.TempDir()

	numLines := 10
	lines := make([]string, 0, 100)
	for i := range numLines {
		lines = append(lines, fmt.Sprintf("This is line %d", i))
	}
	content := strings.Join(lines, "\n")

	testfile := filepath.Join(dir, "subdir", "test.txt")
	os.Mkdir(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(testfile, []byte(content), 0644)
	t.Logf("File created %s", testfile)

	inputs := make(map[string]ReadInput)

	for off := range numLines {
		for lim := range numLines {
			key := fmt.Sprintf("Off%dLim%d", off, lim)
			inputs[key] = ReadInput{"subdir/test.txt", off, lim}
		}
	}

	for name, input := range inputs {
		test := func(t *testing.T) {
			fs := NewLocalFileSystemBackend(dir)
			ctx := context.Background()
			res, err := fs.Read(ctx, input)
			if err != nil {
				t.Error(err)
			}

			expected := strings.Join(lines[input.Offset:min(input.Offset+input.Limit, numLines)], "\n")
			if res.FileContent != expected {
				t.Errorf("File content read \"%s\", want \"%s\"", res.FileContent, expected)
			}
		}
		t.Run(name, test)
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()

	testFile := "test.txt"
	content := "this is some content."

	input := WriteInput{
		Path:    testFile,
		Content: content,
	}

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Write(ctx, input)
	if err != nil {
		t.Error(err)
	}
	if res.Path != "test.txt" {
		t.Errorf("Wrote to %s, want %s", res.Path, "test.txt")
	}

	fileContent, err := os.ReadFile(filepath.Join(dir, testFile))
	if err != nil {
		t.Error(err)
	}

	if string(fileContent) != content {
		t.Errorf("File content read \"%s\", want \"%s\"", fileContent, content)
	}
}

func TestGrep(t *testing.T) {
	dir := t.TempDir()

	content := "this is some content.\nwith multiple lines."
	os.WriteFile(filepath.Join(dir, "one.txt"), []byte(content), 0644)
	os.WriteFile(filepath.Join(dir, "two.txt"), []byte(content), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(filepath.Join(dir, "subdir", "three.txt"), []byte(content), 0644)

	glob := "**/*.txt"
	input := GrepInput{
		Pattern: "^this\\s+is",
		Glob:    &glob,
	}

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Grep(ctx, input)
	if err != nil {
		t.Error(err)
	}

	t.Log(res)
	if len(res.Matches) != 3 {
		t.Errorf("Matched found %d, want %d", len(res.Matches), 3)
	}

	expected := []string{"one.txt", "two.txt", "subdir/three.txt"}
	for i, match := range res.Matches {
		if match.Path != expected[i] {
			t.Errorf("Found %s, want %s", match.Path, expected[i])
		}
	}
}
