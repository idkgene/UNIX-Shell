package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type Shell struct {
	history []string
	historyFile string
	workDir string
}

func NewShell() (*Shell, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	historyFile := filepath.Join(homeDir, ".gosh_history")
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	shell := &Shell{
		historyFile: historyFile,
		workDir: workDir,
	}

	shell.loadHistory()
	return shell, nil
}

func (s *Shell) loadHistory() error {
	file, err := os.OpenFile(s.historyFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s.history = append(s.history, scanner.Text())
	}
	return scanner.Err()
}

func (s *Shell) addToHistory(cmd string) {
	if cmd == "" {
		return
	}
	s.history = append(s.history, cmd)

	f, err := os.OpenFile(s.historyFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		defer f.Close()
		fmt.Fprintln(f, cmd)
	}
}


func (s *Shell) executePipeline(commands [][]string) error {
	var cmds []*exec.Cmd
	for _, cmdArgs:= range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmds = append(cmds, cmd)
	}

	for i := 0; i < len(cmds) - 1; i++ {
		pipe, err := cmds[i].StdoutPipe()
		if err != nil {
			return err
		}
		cmds[i+1].Stdin = pipe
	}

	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[0].Stdin = os.Stdin
	for _, cmd := range cmds {
		cmd.Stderr = os.Stderr
	}

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return err;
		}
	}
	return nil
}

func (s *Shell) parseCommand(input string) [][]string {
	pipeParts := strings.Split(input, "|")
	commands := make([][]string, 0, len(pipeParts))

	for _, part := range pipeParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		commands = append(commands, strings.Fields(part))
	}

	return commands
}

func (s *Shell) executeBuiltin(args []string) (bool, error) {
	switch args[0] {
	case "cd":
		if len(args) < 2 {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return true, err
			}
			args = append(args, homeDir)
		}
		err := os.Chdir(args[1])
		if err != nil {
			return true, err
		}
		s.workDir, _ = os.Getwd()
		return true, nil

	case "exit":
		os.Exit(0)

	case "history":
		for i, cmd := range s.history {
			fmt.Printf("%d: %s\n", i + 1, cmd)
		}
		return true, nil
 }

 	return false, nil
}

func (s *Shell) prompt() string {
	username := os.Getenv("USER")
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s@%s:%s$ ", username, hostname, s.workDir)
}

func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range sigChan {
			fmt.Printf("\nReceived signal: %v\n", sig)
			// maybe some more complex logic later
		}
	}()
}

func main() {
	shell, err := NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize shell: %v\n", err)
		os.Exit(1)
	}

	setupSignalHandler()
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(shell.prompt())

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to read input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		shell.addToHistory(input)

		input = os.ExpandEnv(input)

		commands := shell.parseCommand(input)
		if len(commands) == 0 {
			continue
		}

		if isBuiltin, err := shell.executeBuiltin(commands[0]); isBuiltin {
			if err != nil {
				fmt.Fprintln(os.Stderr, "Builtin failed:", err)
			}
			continue
		}

		err = shell.executePipeline(commands)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to execute pipeline:", err)
		}
	}
}
