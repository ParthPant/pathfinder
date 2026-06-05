package backends

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIgnore(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	lines := []string{"*.txt", "*.db", "subdir/*.csv"}
	content := strings.Join(lines, "\n")

	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(content), 0644)

	p := NewGitIgnorePolicy(dir)

	ignored, _ := p.ShouldIgnore(filepath.Join(dir, "a.txt"))
	assert.True(t, ignored, "a.txt should be ignored.")

	ignored, _ = p.ShouldIgnore("subdir/test.csv")
	assert.True(t, ignored, "subdir/test.csv should be ignored.")

	ignored, _ = p.ShouldIgnore("test.db")
	assert.True(t, ignored, "test.db should be ignored.")

	ignored, _ = p.ShouldIgnore("test.py")
	assert.False(t, ignored, "test.py should NOT be ignored.")
}
