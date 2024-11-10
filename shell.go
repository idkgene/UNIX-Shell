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
		history 			[]string
		historyFile 	string
		workDir 			string
		aliases 			map[string]string
		aliasFile 		string
}

type Command struct {
		Args 					[]string
		Stdin 				string 		// For <
		Stdout 				string 		// For >
		StdoutAppend 	bool 			// For >>
}

func NewShell() (*Shell, error) {
		homeDir, err := os.UserHomeDir()
		
		if err != nil {
			return nil, err
		}

		historyFile := filepath.Join(homeDir, ".gosh_history")
		aliasFile := filepath.Join(homeDir, ".gosh_aliases")


		workDir, err := os.Getwd()
		
		if err != nil {
			return nil, err
		}

		shell := &Shell{
			historyFile: historyFile,
			aliasFile: aliasFile,
			workDir: workDir,
			aliases: make(map[string]string),
		}

		shell.loadHistory()
		shell.loadAliases()
		return shell, nil
}

func (s *Shell) loadAliases() error {
		file, err := os.OpenFile(s.aliasFile, os.O_CREATE|os.O_RDONLY, 0644)
		if err != nil {
			return err
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), "=", 2)
			if len (parts) == 2 {
				s.aliases[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		
		return scanner.Err()
}

func (s *Shell) saveAlias(name, command string) error {
    s.aliases[name] = command
    f, err := os.OpenFile(s.aliasFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    
		if err != nil {
        return err
    }
    
		defer f.Close()
    
    _, err = fmt.Fprintf(f, "%s=%s\n", name, command)
    return err
}

func (s *Shell) parseCommandWithRedirection(input string) ([]Command, error) {
		parts := strings.Split(input, "|")
		commands := make([]Command, 0, len(parts))

		for _, part := range parts {
			cmd := Command{}
			part = strings.TrimSpace(part)

			// >>
			if idx := strings.Index(part, ">>"); idx != -1 {
				cmd.StdoutAppend = true
				cmd.Stdout = strings.TrimSpace(part[idx+2:])
				part = strings.TrimSpace(part[:idx])
			} else if idx := strings.Index(part, ">"); idx != -1 {
				cmd.Stdout = strings.TrimSpace(part[idx+1:])
				part = strings.TrimSpace(part[:idx])
			}

			if idx := strings.Index(part, "<"); idx != -1 {
				cmd.Stdin = strings.TrimSpace(part[idx+1:])
				part = strings.TrimSpace(part[:idx])
		}

			cmd.Args = strings.Fields(part)

			if len(cmd.Args) > 0 {
				if alias, exists := s.aliases[cmd.Args[0]]; exists {
					aliasArgs := strings.Fields(alias)
					cmd.Args = append(aliasArgs, cmd.Args[1:]...)
				}
			}

			if len(cmd.Args) > 0 {
				commands = append(commands, cmd)
			}
		}

		return commands, nil
}

func (s *Shell) executeCommand(cmd Command) error {
		if len(cmd.Args) == 0 {
			return nil
		}

		execCmd := exec.Command(cmd.Args[0], cmd.Args[1:]...)

		if cmd.Stdin != "" {
			file, err := os.Open(cmd.Stdin)
			if err != nil {
				return fmt.Errorf("failed to open input file: %v", err)
			}
			defer file.Close()
			execCmd.Stdin = file
		} else {
			execCmd.Stdin = os.Stdin
		}

		if cmd.Stdout != "" {
			flag := os.O_CREATE | os.O_WRONLY
			if cmd.StdoutAppend {
				flag |= os.O_APPEND
			} else {
				flag |= os.O_TRUNC
			}

			file, err := os.OpenFile(cmd.Stdout, flag, 0644)
			
			if err != nil {
				return fmt.Errorf("failed to open output file: %v", err)
			}
			
			defer file.Close()
			execCmd.Stdout = file
		} else {
			execCmd.Stdout = os.Stdout
		}

		execCmd.Stderr = os.Stderr
		return execCmd.Run()
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


func (s *Shell) executePipeline(commands []Command) error {
		if len(commands) == 1 {
			return s.executeCommand(commands[0])
		}

		var cmds []*exec.Cmd
		
		for _, command:= range commands {
			cmd := exec.Command(command.Args[0], command.Args[1:]...)
			cmds = append(cmds, cmd)
		}

		for i := 0; i < len(cmds) - 1; i++ {
			pipe, err := cmds[i].StdoutPipe()
			
			if err != nil {
				return err
			}
			cmds[i + 1].Stdin = pipe
		}

		if commands[0].Stdin != "" {
			file, err := os.Open(commands[0].Stdin)
			if err != nil {
				return err
			}
			defer file.Close()
			cmds[0].Stdin = file
		} else {
			cmds[0].Stdin = os.Stdin
		}

		lastCmd := commands[len(commands) - 1]
		if lastCmd.Stdout != "" {
			flag := os.O_CREATE | os.O_WRONLY
			if lastCmd.StdoutAppend {
				flag |= os.O_APPEND
			} else {
				flag |= os.O_TRUNC
			}

			file, err := os.OpenFile(lastCmd.Stdout, flag, 0644)
			if err != nil {
				return err
			}
			defer file.Close()
			cmds[len(cmds) - 1].Stdout = file
		} else {
			cmds[len(cmds) - 1].Stdout = os.Stdout
		}

		for _, cmd := range cmds {
			cmd.Stderr = os.Stderr
		}

		for _, cmd := range cmds {
			if err := cmd.Start(); err != nil {
				return err
			}
		}

		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				return err
			}
		}

		return nil
	}

// func (s *Shell) parseCommand(input string) [][]string {
// 		pipeParts := strings.Split(input, "|")
// 		commands := make([][]string, 0, len(pipeParts))

// 		for _, part := range pipeParts {
// 			part = strings.TrimSpace(part)
// 			if part == "" {
// 				continue
// 			}
// 			commands = append(commands, strings.Fields(part))
// 		}

// 		return commands
// }

func (s *Shell) executeBuiltin(cmd Command) (bool, error) {
		switch cmd.Args[0] {
		case "cd":
			if len(cmd.Args) < 2 {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return true, err
				}
				cmd.Args = append(cmd.Args, homeDir)
			}
			err := os.Chdir(cmd.Args[1])
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
	
		case "alias": 
		if len(cmd.Args) == 1 {
			// List all aliases
			for name, command := range s.aliases {
				fmt.Printf("%s=%s\n", name, command)
			}
			return true, nil
		}

		if len(cmd.Args) == 2 {
			if alias, exists := s.aliases[cmd.Args[1]]; exists {
				fmt.Printf("%s=%s\n", cmd.Args[1], alias)
			}
			return true, nil
		}

		name := cmd.Args[1]
		command := strings.Join(cmd.Args[2:], " ")
		err := s.saveAlias(name, command)
		return true, err
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
					fmt.Fprintln(os.Stderr, "Error reading input:", err)
					continue
			}

			input = strings.TrimSpace(input)
			if input == "" {
					continue
			}

			shell.addToHistory(input)
			input = os.ExpandEnv(input)

			commands, err := shell.parseCommandWithRedirection(input)
			if err != nil {
					fmt.Fprintln(os.Stderr, "Error parsing command:", err)
					continue
			}

			if len(commands) == 0 {
					continue
			}

			if isBuiltin, err := shell.executeBuiltin(commands[0]); isBuiltin {
					if err != nil {
							fmt.Fprintln(os.Stderr, "Error:", err)
					}
					continue
			}

			err = shell.executePipeline(commands)
			if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
			}
	}
}
