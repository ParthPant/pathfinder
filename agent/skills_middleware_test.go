package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// writeSkillFile creates a SKILL.md file with the given frontmatter and body
// at the given directory location. The subpath is relative to dir, e.g. "myskill/SKILL.md".
func writeSkillFile(t *testing.T, dir, subpath, frontmatter, body string) string {
	t.Helper()
	fullPath := filepath.Join(dir, subpath)
	err := os.MkdirAll(filepath.Dir(fullPath), 0744)
	require.NoError(t, err)

	content := "---\n" + frontmatter + "\n---\n" + body
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(t, err)
	return fullPath
}

// emptyState returns an AgentState with no system messages.
func emptyState() AgentState {
	return AgentState{
		systemMessages:      nil,
		messages:            nil,
		userRejectedTools:   nil,
	}
}

// stateWithPromptId returns an AgentState that already contains a system
// message carrying the given promptMessageId.
func stateWithPromptId(id string) AgentState {
	return AgentState{
		systemMessages: []messages.Message{
			{
				Role: "system",
				SystemMessage: messages.SystemMessage{
					Id: id,
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// NewSkillsMiddleware
// ---------------------------------------------------------------------------

func TestNewSkillsMiddleware_CreatesDirAndReadsSkills(t *testing.T) {
	dir := t.TempDir()

	// Write a valid skill
	writeSkillFile(t, dir, "math/SKILL.md",
		"name: math\ndescription: Math utilities",
		"Some math skill content.",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)
	assert.Equal(t, dir, m.skillsDir)
	assert.Equal(t, "skill_middleware_sys_prompt", m.promptMessageId)
	assert.Len(t, m.skills, 1)
	assert.Equal(t, "math", m.skills[0].Name)
	assert.Equal(t, "Math utilities", m.skills[0].Description)
	assert.Equal(t, "math/SKILL.md", m.skills[0].Path)
}

func TestNewSkillsMiddleware_CreatesDirEvenIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "skills")
	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)

	_, err = os.Stat(dir)
	assert.NoError(t, err, "directory should have been created")
}

func TestNewSkillsMiddleware_PopulatesDefaultTemplate(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir())
	require.NotNil(t, m)
	assert.NotEmpty(t, m.systemPromptTemplate, "default template must not be empty")
}

func TestNewSkillsMiddleware_WithCustomPromptTemplate(t *testing.T) {
	customTmpl := "Custom: {{.Sources}}"
	m := NewSkillsMiddleware(t.TempDir(), WithSkillsPromptTemplate(customTmpl))
	require.NotNil(t, m)
	assert.Equal(t, customTmpl, m.systemPromptTemplate)
}

func TestNewSkillsMiddleware_SkipsNonSkillFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a SKILL.md
	writeSkillFile(t, dir, "web/SKILL.md",
		"name: web\ndescription: Web research",
		"Do web research.",
	)

	// Write non-SKILL.md files, which should be ignored
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0744)
	os.WriteFile(filepath.Join(dir, "subdir", "notes.md"), []byte("note"), 0644)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)
	assert.Len(t, m.skills, 1, "only SKILL.md files should be loaded")
}

func TestNewSkillsMiddleware_ReadsMultipleSkills(t *testing.T) {
	dir := t.TempDir()

	writeSkillFile(t, dir, "math/SKILL.md",
		"name: math\ndescription: Math utilities",
		"Math content",
	)
	writeSkillFile(t, dir, "web/SKILL.md",
		"name: web\ndescription: Web research",
		"Web content",
	)
	writeSkillFile(t, dir, "code/SKILL.md",
		"name: code\ndescription: Code review",
		"Code content",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)
	assert.Len(t, m.skills, 3)
}

func TestNewSkillsMiddleware_LoadsFileWithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()

	// A SKILL.md without frontmatter is still loaded with empty frontmatter fields
	err := os.MkdirAll(filepath.Join(dir, "nofm"), 0744)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "nofm/SKILL.md"), []byte("Just plain text, no frontmatter."), 0644)
	require.NoError(t, err)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)
	require.Len(t, m.skills, 1)
	s := m.skills[0]
	assert.Equal(t, "nofm/SKILL.md", s.Path)
	assert.Empty(t, s.Name, "name should be empty when no frontmatter")
	assert.Contains(t, s.Content, "Just plain text, no frontmatter.")
}

// ---------------------------------------------------------------------------
// WithSkillsPromptTemplate
// ---------------------------------------------------------------------------

func TestWithSkillsPromptTemplate(t *testing.T) {
	tmpl := "Hello {{.Name}}"
	opt := WithSkillsPromptTemplate(tmpl)

	m := &SkillsMiddleware{}
	opt(m)

	assert.Equal(t, tmpl, m.systemPromptTemplate)
}

// ---------------------------------------------------------------------------
// OnAttach
// ---------------------------------------------------------------------------

func TestOnAttach_ReturnsNil(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir())
	require.NotNil(t, m)

	err := m.OnAttach(nil)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// BeforeAgent
// ---------------------------------------------------------------------------

func TestBeforeAgent_AlreadyHasPrompt(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir())
	require.NotNil(t, m)

	state := stateWithPromptId("skill_middleware_sys_prompt")
	initialCount := len(state.systemMessages)

	newState, err := m.BeforeAgent(context.Background(), nil, nil, state)
	require.NoError(t, err)
	assert.Equal(t, initialCount, len(newState.systemMessages),
		"should not add another prompt when one already exists")
}

func TestBeforeAgent_AddsPromptWhenMissing(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "test/SKILL.md",
		"name: test\ndescription: Test skill",
		"Test content",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)

	state := emptyState()
	newState, err := m.BeforeAgent(context.Background(), nil, nil, state)
	require.NoError(t, err)
	require.Len(t, newState.systemMessages, 1,
		"should add exactly one system message")

	msg := newState.systemMessages[0]
	assert.Equal(t, "system", msg.Role)
	assert.Equal(t, "skill_middleware_sys_prompt", msg.SystemMessage.Id)
	assert.Contains(t, msg.SystemMessage.Content.OfInputText, "test",
		"prompt should reference the loaded skill")
}

func TestBeforeAgent_ErrorOnBadTemplate(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir(), WithSkillsPromptTemplate("{{.BadField"))
	require.NotNil(t, m)

	_, err := m.BeforeAgent(context.Background(), nil, nil, emptyState())
	assert.Error(t, err)
}

func TestBeforeAgent_IdempotentOnMultipleCalls(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "test/SKILL.md",
		"name: test\ndescription: Test skill",
		"Test content",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)

	state := emptyState()

	// First call adds the prompt
	state, err := m.BeforeAgent(context.Background(), nil, nil, state)
	require.NoError(t, err)
	require.Len(t, state.systemMessages, 1)

	// Second call should detect the existing prompt and not add another
	state, err = m.BeforeAgent(context.Background(), nil, nil, state)
	require.NoError(t, err)
	assert.Len(t, state.systemMessages, 1,
		"should not duplicate the system prompt")
}

// ---------------------------------------------------------------------------
// BeforeLlm, AfterAgent, AfterLlm — all pass-through
// ---------------------------------------------------------------------------

func TestPassthroughMethods_ReturnStateUnchanged(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir())
	require.NotNil(t, m)

	ctx := context.Background()
	state := emptyState()

	state1, err := m.BeforeLlm(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state1)

	state2, err := m.AfterAgent(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state2)

	state3, err := m.AfterLlm(ctx, nil, nil, state)
	assert.NoError(t, err)
	assert.Equal(t, state, state3)
}

// ---------------------------------------------------------------------------
// assemblePrompt
// ---------------------------------------------------------------------------

func TestAssemblePrompt_RendersTemplate(t *testing.T) {
	dir := t.TempDir()
	writeSkillFile(t, dir, "math/SKILL.md",
		"name: math\ndescription: Mathematical operations",
		"Math skill content here.",
	)
	writeSkillFile(t, dir, "web/SKILL.md",
		"name: web\ndescription: Web research and crawling",
		"Web skill content here.",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)
	require.Len(t, m.skills, 2)

	result, err := m.assemblePrompt()
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// The rendered output should contain the skill names and descriptions
	assert.Contains(t, result, "math")
	assert.Contains(t, result, "Mathematical operations")
	assert.Contains(t, result, "web")
	assert.Contains(t, result, "Web research and crawling")

	// Should contain the sources (skills dir)
	assert.Contains(t, result, dir)
}

func TestAssemblePrompt_WithCustomTemplate(t *testing.T) {
	customTmpl := "Skills: {{range .Skills}}{{.Name}}: {{.Description}}\n{{end}}"

	m := NewSkillsMiddleware(t.TempDir(),
		WithSkillsPromptTemplate(customTmpl),
	)
	require.NotNil(t, m)

	// Inject a skill manually so we don't depend on file I/O
	m.skills = []Skill{
		{
			Path: "alpha/SKILL.md",
			SkillFrontmatter: SkillFrontmatter{
				Name:        "alpha",
				Description: "Alpha skill",
			},
			Content: "content",
		},
	}

	result, err := m.assemblePrompt()
	require.NoError(t, err)
	assert.Equal(t, "Skills: alpha: Alpha skill\n", result)
}

func TestAssemblePrompt_ErrorOnBadTemplate(t *testing.T) {
	m := &SkillsMiddleware{
		systemPromptTemplate: "{{.NoSuchField}}",
	}

	_, err := m.assemblePrompt()
	assert.Error(t, err)
}

func TestAssemblePrompt_EmptySkills(t *testing.T) {
	m := NewSkillsMiddleware(t.TempDir())
	require.NotNil(t, m)

	// No skills written, so m.skills should be empty
	result, err := m.assemblePrompt()
	require.NoError(t, err)
	assert.NotEmpty(t, result, "should still render even with no skills")
}

// ---------------------------------------------------------------------------
// readSkills (white-box)
// ---------------------------------------------------------------------------

func TestReadSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	skills := readSkills(dir)
	assert.Empty(t, skills)
}

func TestReadSkills_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	skills := readSkills(dir)
	assert.Empty(t, skills, "should not panic on non-existent directory")
}

func TestReadSkills_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()

	// A directory named SKILL.md should be skipped (d.IsDir returns true)
	err := os.MkdirAll(filepath.Join(dir, "SKILL.md"), 0744)
	require.NoError(t, err)

	skills := readSkills(dir)
	assert.Empty(t, skills)
}

func TestReadSkills_ParsesAllFrontmatterFields(t *testing.T) {
	dir := t.TempDir()

	writeSkillFile(t, dir, "full/SKILL.md",
		"name: full\n"+
			"description: A full-featured skill\n"+
			"model: gpt-4\n"+
			"allowed-tools: Read Write\n"+
			"disable-model-invocation: true\n"+
			"user-invocable: true",
		"Body content here.",
	)

	skills := readSkills(dir)
	require.Len(t, skills, 1)

	s := skills[0]
	assert.Equal(t, "full/SKILL.md", s.Path)
	assert.Equal(t, "full", s.Name)
	assert.Equal(t, "A full-featured skill", s.Description)
	assert.Equal(t, "gpt-4", s.Model)
	assert.Equal(t, "Read Write", s.AllowedTools)
	assert.True(t, s.DisableModelInvocation)
	assert.True(t, s.UserInvocable)
	assert.Contains(t, s.Content, "Body content here.")
}

func TestReadSkills_PreservesSkillContent(t *testing.T) {
	dir := t.TempDir()

	body := "# Math Skill\n\nDo math operations.\n\n```go\nfunc Add(a, b int) int { return a + b }\n```\n"
	writeSkillFile(t, dir, "math/SKILL.md",
		"name: math\ndescription: Math",
		body,
	)

	skills := readSkills(dir)
	require.Len(t, skills, 1)
	assert.Contains(t, skills[0].Content, "Do math operations.")
}

func TestReadSkills_WithPartialFrontmatter(t *testing.T) {
	dir := t.TempDir()

	// Only name, no description
	writeSkillFile(t, dir, "minimal/SKILL.md",
		"name: minimal",
		"Just content.",
	)

	skills := readSkills(dir)
	require.Len(t, skills, 1)
	assert.Equal(t, "minimal", skills[0].Name)
	assert.Empty(t, skills[0].Description)
	assert.False(t, skills[0].DisableModelInvocation)
	assert.False(t, skills[0].UserInvocable)
}

// ---------------------------------------------------------------------------
// Integration: end-to-end BeforeAgent with real template
// ---------------------------------------------------------------------------

func TestIntegration_BeforeAgentProducesValidPrompt(t *testing.T) {
	dir := t.TempDir()

	writeSkillFile(t, dir, "math/SKILL.md",
		"name: math\ndescription: Math utilities",
		"Math content",
	)
	writeSkillFile(t, dir, "web/SKILL.md",
		"name: web\ndescription: Web research",
		"Web content",
	)

	m := NewSkillsMiddleware(dir)
	require.NotNil(t, m)

	state, err := m.BeforeAgent(context.Background(), nil, nil, emptyState())
	require.NoError(t, err)
	require.Len(t, state.systemMessages, 1)

	promptText := state.systemMessages[0].SystemMessage.Content.OfInputText

	// The default template lists skill names with "Name:" prefix
	assert.Contains(t, promptText, "Name: math")
	assert.Contains(t, promptText, "Name: web")
	assert.Contains(t, promptText, "Math utilities")
	assert.Contains(t, promptText, "Web research")
	assert.Contains(t, promptText, "math/SKILL.md")
	assert.Contains(t, promptText, "web/SKILL.md")

	// Template should contain progressive disclosure instructions
	assert.Contains(t, promptText, "Progressive Disclosure")
	assert.Contains(t, promptText, "Recognize when a skill applies")
}

// ---------------------------------------------------------------------------
// Benchmark: reading and assembling
// ---------------------------------------------------------------------------

func BenchmarkReadSkills(b *testing.B) {
	dir := b.TempDir()

	for i := range 100 {
		skillDir := filepath.Join(dir, "skill-"+strings.Repeat("0", 3-len(itoa(i)))+itoa(i))
		writeSkillFile(&testing.T{}, skillDir, "SKILL.md",
			"name: skill-"+itoa(i)+"\ndescription: Description for skill "+itoa(i),
			"Content for skill "+itoa(i),
		)
	}
	b.ResetTimer()

	for b.Loop() {
		_ = readSkills(dir)
	}
}

func BenchmarkAssemblePrompt(b *testing.B) {
	dir := b.TempDir()

	for i := range 20 {
		skillDir := filepath.Join(dir, "skill-"+itoa(i))
		writeSkillFile(&testing.T{}, skillDir, "SKILL.md",
			"name: skill-"+itoa(i)+"\ndescription: Description",
			"Content",
		)
	}

	m := NewSkillsMiddleware(dir)

	b.ResetTimer()
	for b.Loop() {
		_, _ = m.assemblePrompt()
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// itoa is a simple int-to-string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}