package agent

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ParthPant/pathfinder/messages"
	"github.com/ParthPant/pathfinder/prompts"
	"github.com/adrg/frontmatter"
)

type SkillsMiddleware struct {
	skillsDir            string
	systemPromptTemplate string
	promptMessageId      string
	skills               []Skill
}

type SkillsOpt = func(*SkillsMiddleware)

type SkillFrontmatter struct {
	Name                   string `yaml:"name"`
	Description            string `yaml:"description"`
	Model                  string `yaml:"model"`
	AllowedTools           string `yaml:"allowed-tools"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation"`
	UserInvocable          bool   `yaml:"user-invocable"`
}

type Skill struct {
	Path string
	SkillFrontmatter
	Content string
}

func NewSkillsMiddleware(skillsDir string, opts ...SkillsOpt) *SkillsMiddleware {
	if err := os.MkdirAll(skillsDir, 0744); err != nil {
		slog.Error("Error creating skills directory", "error", err)
		return nil
	}

	m := SkillsMiddleware{
		skillsDir:            skillsDir,
		promptMessageId:      "skill_middleware_sys_prompt",
		systemPromptTemplate: prompts.SkillsPrompt,
	}

	for _, o := range opts {
		o(&m)
	}

	for _, skill := range readSkills(skillsDir) {
		m.skills = append(m.skills, skill)
	}

	return &m
}

func WithSkillsPromptTemplate(t string) SkillsOpt {
	return func(m *SkillsMiddleware) {
		m.systemPromptTemplate = t
	}
}

func (m *SkillsMiddleware) OnAttach(a *Agent) error {
	return nil
}

func (m *SkillsMiddleware) BeforeAgent(ctx context.Context, eventCh AgentEventCh, intrCh AgentIntrCh, state AgentState) (AgentState, error) {
	for _, sysMessage := range state.systemMessages {
		if sysMessage.SystemMessage.Id == m.promptMessageId {
			slog.Debug("Skills prompt already present in the conversation. Skipping retrieving memories.")
			return state, nil
		}
	}

	if prompt, err := m.assemblePrompt(); err != nil {
		slog.Error("Error while creating skills system prompt", "error", err)
		return state, err
	} else {
		slog.Debug("Adding Skills Prompt to the agent's system prompts.")
		state.systemMessages = append(state.systemMessages, messages.NewTextMessage("system", prompt, &m.promptMessageId))
		return state, nil
	}
}

func (m *SkillsMiddleware) AfterAgent(ctx context.Context, eventCh AgentEventCh, intrCh AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SkillsMiddleware) BeforeLlm(ctx context.Context, eventCh AgentEventCh, intrCh AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SkillsMiddleware) AfterLlm(ctx context.Context, eventCh AgentEventCh, intrCh AgentIntrCh, state AgentState) (AgentState, error) {
	return state, nil
}

func (m *SkillsMiddleware) assemblePrompt() (string, error) {
	var input struct {
		Sources []string
		Skills  []Skill
	}
	input.Sources = []string{m.skillsDir}
	input.Skills = m.skills

	t, err := template.New("sys_prompt").Parse(m.systemPromptTemplate)
	if err != nil {
		return "", err
	}

	w := strings.Builder{}
	err = t.Execute(&w, input)
	if err != nil {
		return "", err
	}
	return w.String(), nil
}

func readSkills(dir string) []Skill {
	skills := make([]Skill, 0)

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			slog.Error("Error while walking directory", "error", e)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() != "SKILL.md" {
			return nil
		}

		slog.Debug("Discovered Skill", "path", path)
		skillPath := filepath.Join(dir, path)
		file, err := os.Open(skillPath)
		if err != nil {
			slog.Error("Error while reading skill file", "path", path, "error", err)
			return nil
		}

		var matter SkillFrontmatter
		content, err := frontmatter.Parse(file, &matter)
		if err != nil {
			slog.Warn("Error while parsing Skill Frontmatter", "path", path, "error", err)
			return nil
		}

		skills = append(skills, Skill{
			Path:             path,
			SkillFrontmatter: matter,
			Content:          string(content),
		})

		return nil
	})

	if err != nil {
		slog.Error("Error while exploring Skills directory", "path", dir, "error", err.Error())
	}

	return skills
}
