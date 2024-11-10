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

func main() {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
        os.Exit(1)
    }

    config := &Config{
        HistoryFile: filepath.Join(homeDir, ".gosh_history"),
        AliasFile:   filepath.Join(homeDir, ".gosh_aliases"),
    }

    shell := &Shell{
        config:  config,
        workDir: homeDir,
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        for sig := range sigChan {
            fmt.Printf("\nReceived signal: %v\n", sig)
            if sig == syscall.SIGTERM {
                os.Exit(0)
            }
            fmt.Print(shell.prompt())
        }
    }()

    reader := bufio.NewReader(os.Stdin)

    for {
        fmt.Print(shell.prompt())

        input, err := reader.ReadString('\n')
        if err != nil {
            if err.Error() == "EOF" {
                fmt.Println("\nExiting...")
                break
            }
            fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
            continue
        }

        input = strings.TrimSpace(input)
        if input == "" {
            continue
        }

        if err := shell.Execute(input); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        }
    }
}

type Shell struct {
    config  *Config
    workDir string
}

type Config struct {
    HistoryFile string
    AliasFile   string
}

func (s *Shell) prompt() string {
    return fmt.Sprintf("%s $ ", s.workDir)
}

func (s *Shell) Execute(input string) error {
    args := strings.Fields(input)
    
    if len(args) == 0 {
        return nil
    }

    switch args[0] {
    case "cd":
        if len(args) < 2 {
            homeDir, err := os.UserHomeDir()
            if err != nil {
                return err
            }
            return os.Chdir(homeDir)
        }
        return os.Chdir(args[1])
    
    case "exit":
        os.Exit(0)
    }

    cmd := exec.Command(args[0], args[1:]...)
    
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}
