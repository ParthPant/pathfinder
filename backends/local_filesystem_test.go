package backends

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
	for _, match := range out.Matches {
		t.Log(match)
	}

	if len(out.Matches) != 3 {
		t.Errorf("Found %d, Expected %d", len(out.Matches), 3)
	}
}
