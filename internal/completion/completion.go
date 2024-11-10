package completion

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gosh/internal/alias"
	"gosh/internal/history"
	"sort"
)

type Shell struct {
	aliases    *alias.Manager
	completion *Manager
	workDir    string
	history    *history.Manager
}

type Manager struct {
    shell        ShellInterface
    suggestions  map[string][]string
    customComps  map[string]CompletionFunc
}

type CompletionFunc func(args []string) []string

type ShellInterface interface {
    GetAliases() map[string]string
    GetHistory() []string
    GetWorkDir() string
}

func (s *Shell) GetAliases() map[string]string {
	return s.aliases.GetAll()
}

func (s *Shell) GetHistory() []string {
	return s.history.GetAll()
}

func (s *Shell) GetWorkDir() string {
	return s.workDir
}

func NewManager(shell ShellInterface) *Manager {
    return &Manager{
        shell:       shell,
        suggestions: make(map[string][]string),
        customComps: make(map[string]CompletionFunc),
    }
}

func (m *Manager) Initialize() error {
    m.suggestions["cd"] = []string{"../", "./"}
    m.suggestions["git"] = []string{"status", "commit", "push", "pull", "checkout", "branch"}
    
    m.customComps["cd"] = m.completePath
    m.customComps["git"] = m.completeGit

    return nil
}

func (m *Manager) Complete(line string, pos int) []string {
    words := strings.Fields(line[:pos])
		if len(words) == 0 {
			return m.completeCommand("")
	}

	if len(words) == 1 {
			return m.completeCommand(words[0])
	}

	if completer, exists := m.customComps[words[0]]; exists {
			return completer(words[1:])
	}

	return m.completePath(words[len(words)-1])
}

func (m *Manager) completeCommand(prefix string) []string {
	var completions []string

	builtins := []string{"cd", "exit", "history", "alias", "export", "echo", "pwd"}
	for _, cmd := range builtins {
			if strings.HasPrefix(cmd, prefix) {
					completions = append(completions, cmd)
			}
	}

	for alias := range m.shell.GetAliases() {
			if strings.HasPrefix(alias, prefix) {
					completions = append(completions, alias)
			}
	}

	pathDirs := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, dir := range pathDirs {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
					continue
			}
			for _, file := range files {
					if strings.HasPrefix(file.Name(), prefix) && !file.IsDir() {
							if file.Mode()&0111 != 0 {
									completions = append(completions, file.Name())
							}
					}
			}
	}

	return uniqueStrings(completions)
}

func uniqueStrings(strs []string) []string {
	keys := make(map[string]bool)
	var list []string

	for _, entry := range strs {
			if _, value := keys[entry]; !value {
					keys[entry] = true
					list = append(list, entry)
			}
	}

		sort.Strings(list)
		return list
}

func (m *Manager) completePath(prefix string) []string {
	var completions []string
	
	basePath := "."
	searchPrefix := prefix
	
	if strings.Contains(prefix, "/") {
			basePath = filepath.Dir(prefix)
			searchPrefix = filepath.Base(prefix)
	}

	files, err := ioutil.ReadDir(basePath)
	if err != nil {
			return completions
	}

	for _, file := range files {
			name := file.Name()
			if strings.HasPrefix(name, searchPrefix) {
					if file.IsDir() {
							name += "/"
					}
					if basePath != "." {
							name = filepath.Join(filepath.Dir(prefix), name)
					}
					completions = append(completions, name)
			}
	}

	return completions
}

func (m *Manager) completeGit(args []string) []string {
	if len(args) == 0 {
			return m.suggestions["git"]
	}

	gitCompletions := map[string][]string{
			"checkout": m.getGitBranches(),
			"branch":   m.getGitBranches(),
			"merge":    m.getGitBranches(),
	}

	if completions, exists := gitCompletions[args[0]]; exists {
			return completions
	}

	return nil
}

func (m *Manager) getGitBranches() []string {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
			return nil
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	return branches
}
