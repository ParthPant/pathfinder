package backends

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Ls tests
// ---------------------------------------------------------------------------

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

func TestLs_Subdir(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(filepath.Join(dir, "subdir", "alpha.txt"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "subdir", "beta.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Ls(ctx, LsInput{Path: "subdir"})
	require.NoError(t, err)

	names := make([]string, len(res.Entries))
	for i, e := range res.Entries {
		names[i] = e.Name
	}
	slices.Sort(names)

	expected := []string{"alpha.txt", "beta.txt"}
	assert.Equal(t, expected, names)
}

func TestLs_NonExistentDir(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Ls(ctx, LsInput{Path: "nonexistent"})
	require.Error(t, err)
}

func TestLs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Ls(ctx, LsInput{Path: "/"})
	require.NoError(t, err)
	assert.Empty(t, res.Entries)
}

// ---------------------------------------------------------------------------
// Glob tests
// ---------------------------------------------------------------------------

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

func TestGlob_NoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	path := "/"
	res, err := fs.Glob(ctx, GlobInput{Pattern: "*.py", Path: &path})
	require.NoError(t, err)
	assert.Empty(t, res.Matches)
}

func TestGlob_DefaultRoot(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Glob(ctx, GlobInput{Pattern: "*.txt"})
	require.NoError(t, err)
	assert.Len(t, res.Matches, 1)
	assert.Equal(t, "a.txt", res.Matches[0].Name)
}

func TestGlob_IgnoresIgnoredFiles(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{"*.log"}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)
	os.WriteFile(filepath.Join(dir, "app.log"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "app.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	path := "/"
	res, err := fs.Glob(ctx, GlobInput{Pattern: "*.*", Path: &path})
	require.NoError(t, err)

	// .gitignore, app.txt — app.log should be ignored
	names := make([]string, len(res.Matches))
	for i, m := range res.Matches {
		names[i] = m.Name
	}
	assert.Contains(t, names, "app.txt")
	assert.NotContains(t, names, "app.log")
}

// ---------------------------------------------------------------------------
// Read tests
// ---------------------------------------------------------------------------

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
				t.Errorf("File content read %q, want %q", res.FileContent, expected)
			}
		}
		t.Run(name, test)
	}
}

func TestRead_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Read(ctx, ReadInput{Path: "nope.txt", Limit: 10})
	require.Error(t, err)
}

func TestRead_IgnoredFile(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{"secret.key"}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)
	os.WriteFile(filepath.Join(dir, "secret.key"), []byte("sensitive data"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Read(ctx, ReadInput{Path: "secret.key", Limit: 10})
	require.Error(t, err)
	assert.IsType(t, FileSystemError{}, err)
}

func TestRead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "empty.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Read(ctx, ReadInput{Path: "empty.txt", Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, res.FileContent)
	assert.Equal(t, "empty.txt", res.FileInfo.Name)
}

// ---------------------------------------------------------------------------
// Write tests
// ---------------------------------------------------------------------------

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
		t.Errorf("File content read %q, want %q", fileContent, content)
	}
}

func TestWrite_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("old content"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Write(ctx, WriteInput{Path: "existing.txt", Content: "new content"})
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "existing.txt"))
	require.NoError(t, err)
	assert.Equal(t, "new content", string(data))
}

func TestWrite_IgnoredFile(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{"*.db"}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Write(ctx, WriteInput{Path: "data.db", Content: "forbidden"})
	require.Error(t, err)
	assert.IsType(t, FileSystemError{}, err)

	// File should NOT exist
	_, err = os.Stat(filepath.Join(dir, "data.db"))
	assert.True(t, os.IsNotExist(err))
}

func TestWrite_CreatesNestedDir(t *testing.T) {
	dir := t.TempDir()

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	// os.WriteFile does NOT create parent dirs, so writing to a nested path
	// should fail — we test that the error is surfaced.
	_, err := fs.Write(ctx, WriteInput{Path: "nested/file.txt", Content: "hello"})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Grep tests
// ---------------------------------------------------------------------------

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

func TestGrep_NoGlob_RecursiveDir(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\nfoo bar\n"), 0644)
	os.Mkdir(filepath.Join(dir, "inner"), 0744)
	os.WriteFile(filepath.Join(dir, "inner", "b.txt"), []byte("hello again\n"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	// No glob — Grep on the root directory should recurse and search all files
	res, err := fs.Grep(ctx, GrepInput{Pattern: "hello", Path: "/"})
	require.NoError(t, err)
	assert.Len(t, res.Matches, 2)
}

func TestGrep_SingleFile(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "data.txt"), []byte("apple\nbanana\napple\ncherry\n"), 0644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("nothing\n"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Grep(ctx, GrepInput{Pattern: "apple", Path: "/data.txt"})
	require.NoError(t, err)
	assert.Len(t, res.Matches, 2)
	for _, m := range res.Matches {
		assert.Equal(t, "data.txt", m.Path)
	}
}

func TestGrep_NoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.txt"), []byte("aaa\nbbb\n"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Grep(ctx, GrepInput{Pattern: "zzz", Path: "/"})
	require.NoError(t, err)
	assert.Empty(t, res.Matches)
}

func TestGrep_IgnoresIgnoredFiles(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{"*.secret"}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)
	os.WriteFile(filepath.Join(dir, "good.txt"), []byte("sensitive data"), 0644)
	os.WriteFile(filepath.Join(dir, "bad.secret"), []byte("sensitive data"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Grep(ctx, GrepInput{Pattern: "sensitive", Path: "/"})
	require.NoError(t, err)
	assert.Len(t, res.Matches, 1)
	assert.Equal(t, "good.txt", res.Matches[0].Path)
}

func TestGrep_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Grep(ctx, GrepInput{Pattern: "[invalid", Path: "/"})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Edit tests (previously untested!)
// ---------------------------------------------------------------------------

func TestEdit_SingleReplace(t *testing.T) {
	dir := t.TempDir()
	original := "hello world, hello universe"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(original), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	input := EditInput{
		Path:      "test.txt",
		OldString: "hello",
		NewString: "goodbye",
	}

	res, err := fs.Edit(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", res.Path)
	assert.Equal(t, 2, res.Occurrences)

	data, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "goodbye world, hello universe", string(data))
}

func TestEdit_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	original := "a|a|a"
	os.WriteFile(filepath.Join(dir, "list.txt"), []byte(original), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	input := EditInput{
		Path:       "list.txt",
		OldString:  "a",
		NewString:  "b",
		ReplaceAll: true,
	}

	res, err := fs.Edit(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, 3, res.Occurrences)

	data, err := os.ReadFile(filepath.Join(dir, "list.txt"))
	require.NoError(t, err)
	assert.Equal(t, "b|b|b", string(data))
}

func TestEdit_NoMatch(t *testing.T) {
	dir := t.TempDir()
	original := "some content"
	os.WriteFile(filepath.Join(dir, "doc.txt"), []byte(original), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	input := EditInput{
		Path:      "doc.txt",
		OldString: "nonexistent",
		NewString: "replacement",
	}

	res, err := fs.Edit(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Occurrences)

	// File content should remain unchanged
	data, err := os.ReadFile(filepath.Join(dir, "doc.txt"))
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestEdit_NonExistentFile(t *testing.T) {
	dir := t.TempDir()

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Edit(ctx, EditInput{
		Path:      "missing.txt",
		OldString: "old",
		NewString: "new",
	})
	require.Error(t, err)
}

func TestEdit_IgnoredFile(t *testing.T) {
	dir := t.TempDir()

	ignoreLines := []string{"*.config"}
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(strings.Join(ignoreLines, "\n")), 0644)
	os.WriteFile(filepath.Join(dir, "app.config"), []byte("key=value"), 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	_, err := fs.Edit(ctx, EditInput{
		Path:      "app.config",
		OldString: "key",
		NewString: "setting",
	})
	require.Error(t, err)
	assert.IsType(t, FileSystemError{}, err)

	// File should remain unchanged
	data, err := os.ReadFile(filepath.Join(dir, "app.config"))
	require.NoError(t, err)
	assert.Equal(t, "key=value", string(data))
}

func TestEdit_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "empty.txt"), []byte{}, 0644)

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	res, err := fs.Edit(ctx, EditInput{
		Path:      "empty.txt",
		OldString: "something",
		NewString: "else",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Occurrences)
}

// ---------------------------------------------------------------------------
// Round-trip integration: Write -> Read -> Edit -> Read
// ---------------------------------------------------------------------------

func TestWriteReadEditRoundTrip(t *testing.T) {
	dir := t.TempDir()

	fs := NewLocalFileSystemBackend(dir)
	ctx := context.Background()

	// Write
	_, err := fs.Write(ctx, WriteInput{Path: "roundtrip.txt", Content: "line1\nline2\nline3"})
	require.NoError(t, err)

	// Read
	res, err := fs.Read(ctx, ReadInput{Path: "roundtrip.txt", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\nline3", res.FileContent)

	// Edit (replace all)
	_, err = fs.Edit(ctx, EditInput{
		Path:       "roundtrip.txt",
		OldString:  "line",
		NewString:  "row",
		ReplaceAll: true,
	})
	require.NoError(t, err)

	// Read again
	res, err = fs.Read(ctx, ReadInput{Path: "roundtrip.txt", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, "row1\nrow2\nrow3", res.FileContent)
}

// ---------------------------------------------------------------------------
// NewLocalFileSystemBackend
// ---------------------------------------------------------------------------

func TestNewLocalFileSystemBackend_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFileSystemBackend(dir)

	assert.Equal(t, dir, fs.root)
	assert.NotNil(t, fs)
	assert.NotNil(t, fs.ignorePolicy)
}

func TestNewLocalFileSystemBackend_RelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fs := NewLocalFileSystemBackend(".")
	assert.NotNil(t, fs)
	assert.True(t, filepath.IsAbs(fs.root))
}

func TestNewLocalFileSystemBackend_MissingDirStillWorks(t *testing.T) {
	// The backend doesn't require the working dir to exist at creation time
	fs := NewLocalFileSystemBackend("/tmp/pathfinder-nonexistent-test-12345")
	assert.NotNil(t, fs)
}
