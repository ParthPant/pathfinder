package prompts

import (
	_ "embed"
)

//go:embed memory.txt
var MemoryPrompt string

//go:embed summarization.txt
var SummarizationPrompt string

//go:embed skills.txt
var SkillsPrompt string
