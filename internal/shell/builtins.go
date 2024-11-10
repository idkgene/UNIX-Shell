package shell

import (
    "fmt"
    "os"
    "sort"
    "strings"
)

type BuiltinCommand struct {
    Name        		string
    Description 		string
    Execute     		func(s *Shell, args []string) error
}

var builtinCommands = map[string]BuiltinCommand{
    "cd": {
        Name:        		"cd",
        Description: 		"Change current directory",
        Execute:     		cdCommand,
    },
    "exit": {
        Name:        		"exit",
        Description: 		"Exit the shell",
        Execute:     		exitCommand,
    },
    "alias": {
        Name:        		"alias",
        Description: 		"Define or display aliases",
        Execute:     		aliasCommand,
    },
    "history": { 
        Name:        		"history",
        Description: 		"Display command history",
        Execute:     		historyCommand,
    },
    "help": {
        Name:        		"help",
        Description: 		"Display help for built-in commands",
        Execute:     		helpCommand,
    },
    "export": {
        Name:        		"export",
        Description: 		"Set environment variables",
        Execute:     		exportCommand,
    },
    "source": {
        Name:        		"source",
        Description: 		"Execute commands from a file",
        Execute:     		sourceCommand,
    },
}

func cdCommand(s *Shell, args []string) error {
    var dir string
    if len(args) < 2 {
        var err error
        dir, err = os.UserHomeDir()
        if err != nil {
            return err
        }
    } else {
        dir = args[1]
    }

    if err := os.Chdir(dir); err != nil {
        return err
    }

    newDir, err := os.Getwd()
    if err != nil {
        return err
    }
    s.workDir = newDir
    return nil
}

func exitCommand(s *Shell, args []string) error {
    s.Stop()
    os.Exit(0)
    return nil
}

func aliasCommand(s *Shell, args []string) error {
	if len(args) == 1 {
			aliases := s.aliases.GetAll()
			var sortedAliases []string
			for name, cmd := range aliases {
					sortedAliases = append(sortedAliases, fmt.Sprintf("%s='%s'", name, cmd))
			}
			sort.Strings(sortedAliases)
			for _, alias := range sortedAliases {
					fmt.Println(alias)
			}
			return nil
	}

	if len(args) >= 2 {
			parts := strings.SplitN(args[1], "=", 2)
			if len(parts) != 2 {
					return fmt.Errorf("invalid alias format. Use: alias name=command")
			}
			name := strings.TrimSpace(parts[0])
			command := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			return s.aliases.Add(name, command)
	}
	return nil
}


func historyCommand(s *Shell, args []string) error {
		entries := s.history.GetAll()
		if len(args) > 1 {
				count := 0
				if _, err := fmt.Sscanf(args[1], "%d", &count); err == nil {
						if count > 0 && count < len(entries) {
								entries = entries[len(entries)-count:]
						}
				}
		}
			
		for i, entry := range entries {
				fmt.Printf("%5d  %s\n", i+1, entry)
		}
				return nil
}
	
func helpCommand(s *Shell, args []string) error {
		if len(args) > 1 {
				if cmd, exists := builtinCommands[args[1]]; exists {
						fmt.Printf("%s - %s\n", cmd.Name, cmd.Description)
						return nil
				}
				
				return fmt.Errorf("no help available for '%s'", args[1])
			}
	
		fmt.Println("Built-in commands:")
		var names []string
		for name := range builtinCommands {
				names = append(names, name)
		}

		sort.Strings(names)
			
		for _, name := range names {
				fmt.Printf("  %-10s - %s\n", name, builtinCommands[name].Description)
		}

		return nil
}
	
	func exportCommand(s *Shell, args []string) error {
			if len(args) < 2 {
					for _, env := range os.Environ() {
							fmt.Println(env)
					}
					return nil
			}
	
			for _, arg := range args[1:] {
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) != 2 {
							return fmt.Errorf("invalid export format: %s", arg)
					}
					if err := os.Setenv(parts[0], parts[1]); err != nil {
							return err
					}
			}
			return nil
	}
	
	func sourceCommand(s *Shell, args []string) error {
			if len(args) < 2 {
					return fmt.Errorf("source: filename argument required")
			}
	
			filename := args[1]
			content, err := os.ReadFile(filename)
			if err != nil {
					return err
			}
	
			commands := strings.Split(string(content), "\n")
			for _, cmd := range commands {
					cmd = strings.TrimSpace(cmd)
					if cmd == "" || strings.HasPrefix(cmd, "#") {
							continue
					}
					if err := s.Execute(cmd); err != nil {
							return fmt.Errorf("source: error executing '%s': %v", cmd, err)
					}
			}
			return nil
	}	
