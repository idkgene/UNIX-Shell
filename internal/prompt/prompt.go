package shell

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gosh/internal/config"
)

type Shell struct {
	config *config.Config
	workDir string
}

type PromptBuilder struct {
    shell *Shell
}

func NewPromptBuilder(s *Shell) *PromptBuilder {
    return &PromptBuilder{shell: s}
}

func (p *PromptBuilder) Build() string {
    prompt := p.shell.config.Prompt
    
    replacements := map[string]func() string{
        "\\u": func() string { return os.Getenv("USER") },
        "\\h": func() string { hostname, _ := os.Hostname(); return hostname },
        "\\w": func() string { return p.shell.workDir },
        "\\W": func() string { return filepath.Base(p.shell.workDir) },
        "\\t": func() string { return time.Now().Format("13:13:13") },
        "\\$": func() string { 
            if os.Geteuid() == 0 {
                return "#"
            }
            return "$"
        },
        "\\v": func() string { return "0.0.0" },
    }

    for pattern, replacer := range replacements {
        prompt = strings.ReplaceAll(prompt, pattern, replacer())
    }

    if p.shell.config.ColorScheme.Prompt != "" {
        prompt = p.shell.config.ColorScheme.Prompt + prompt + "\033[0m"
    }

    return prompt
}
