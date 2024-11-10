package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"gosh/internal/alias"
	"gosh/internal/completion"
	"gosh/internal/config"
	"gosh/internal/history"
	"gosh/internal/plugins"
)

type Shell struct {
	config     *config.Config
	history    *history.Manager
	aliases    *alias.Manager
	completion *completion.Manager
	parser     *Parser
	executor   *Executor
	workDir    string
	
	processGroup sync.Map
	sigChan     chan os.Signal
	stopChan    chan struct{}
	
	interactive bool
	lastExitCode int
}

type ShellOption func(*Shell) error

func NewShell(opts ...ShellOption) (*Shell, error) {
    cfg, err := config.Load()
    
		if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    workDir, err := os.Getwd()
    
		if err != nil {
        return nil, fmt.Errorf("failed to get working directory: %w", err)
    }

    s := &Shell{
			config:      cfg,
			workDir:     workDir,
			sigChan:     make(chan os.Signal, 1),
			stopChan:    make(chan struct{}),
			interactive: true,
	}

		s.history, err = history.NewManager(cfg.HistoryFile)
		
		if err != nil {
			return nil, fmt.Errorf("failed to initialize history: %w", err)
		}

		s.aliases = alias.NewManager(cfg.AliasFile)

		s.completion = completion.NewManager(s)

		s.parser = NewParser(s)
		s.executor = NewExecutor(s)

		for _, opt := range opts {
			if err := opt(s); err != nil {
				return nil, err
			}
		}
	
		return s, nil
}

func (s *Shell) Start() error {
	signal.Notify(s.sigChan, syscall.SIGINT, syscall.SIGTERM)
	go s.handleSignals()

	if err := s.initialize(); err != nil {
			return err
	}


	return s.loop()
}

func (s *Shell) handleSignals() {
	for sig := range s.sigChan {
			switch sig {
			case syscall.SIGINT:
					fmt.Println("\n^C")

					fmt.Print(s.getPrompt())
			case syscall.SIGTERM:
					s.Stop()
			}
	}
}

func (s *Shell) getPrompt() string {
	return fmt.Sprintf("%s $ ", s.workDir)
}

func (s *Shell) Stop() error {
    close(s.stopChan)
    // Cleanup
    return s.cleanup()
}

func (s *Shell) initialize() error {
    if err := s.initializeEnvironment(); err != nil {
        return err
    }

    if err := s.loadPlugins(); err != nil {
        return err
    }

    if err := s.completion.Initialize(); err != nil {
        return err
    }
	
		return nil
}

func (s *Shell) initializeEnvironment() error {
	env := map[string]string{
			"GOSH_VERSION": "1.0.0",
			"GOSH_PATH":    os.Args[0],
			"SHELL":        os.Args[0],
	}

	for k, v := range env {
			if err := os.Setenv(k, v); err != nil {
					return fmt.Errorf("failed to set env %s: %w", k, err)
			}
	}
	
	return nil
}

func (s *Shell) cleanup() error {
	if err := s.history.Save(); err != nil {
			return fmt.Errorf("failed to save history: %w", err)
	}

	if err := s.aliases.Save(); err != nil {
			return fmt.Errorf("failed to save aliases: %w", err)
	}

	return nil
}

func (s *Shell) loop() error {
	reader := bufio.NewReader(os.Stdin)
	
	for {
			select {
			case <-s.stopChan:
					return nil
			default:
					fmt.Print(s.getPrompt())
					
					input, err := reader.ReadString('\n')
					if err != nil {
							if err == io.EOF {
									return nil
							}
							return err
					}

					input = strings.TrimSpace(input)
					if input == "" {
							continue
					}

					if err := s.Execute(input); err != nil {
							fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					}
			}
	}
}

func (s *Shell) Execute(input string) error {
	s.history.Add(input)

	commands, err := s.parser.Parse(input)
	
	if err != nil {
			return err
	}

	for _, cmd := range commands {
			if builtin, ok := builtinCommands[cmd.Args[0]]; ok {
					if err := builtin.Execute(s, cmd.Args); err != nil {
							return err
					}
					continue
			}

			if err := s.executor.Execute(context.Background(), []Command{cmd}); err != nil {
					return err
			}
	}

	return nil
}

// func (s *Shell) setupSignalHandling() {
// 	signal.Notify(s.sigChan,
// 			syscall.SIGINT,
// 			syscall.SIGTERM,
// 			syscall.SIGTSTP,
// 			syscall.SIGCONT)

// 	go func() {
// 			for sig := range s.sigChan {
// 					switch sig {
// 					case syscall.SIGINT:
// 							s.processGroup.Range(func(key, value interface{}) bool {
// 									pg := value.(*ProcessGroup)
// 									syscall.Kill(-pg.Pgid, syscall.SIGINT)
// 									return true
// 							})
// 					case syscall.SIGTERM:
// 							s.Stop()
// 					case syscall.SIGTSTP:
// 							s.processGroup.Range(func(key, value interface{}) bool {
// 									pg := value.(*ProcessGroup)
// 									syscall.Kill(-pg.Pgid, syscall.SIGTSTP)
// 									return true
// 							})
// 					case syscall.SIGCONT:
// 							s.processGroup.Range(func(key, value interface{}) bool {
// 									pg := value.(*ProcessGroup)
// 									syscall.Kill(-pg.Pgid, syscall.SIGCONT)
// 									return true
// 							})
// 					}
// 			}
// 	}()
// }

func (s *Shell) loadPlugins() error {
	if !s.config.PluginsEnabled {
			return nil
	}

	pluginManager := plugins.NewManager(s.config.PluginsDir)
	return pluginManager.LoadPlugins()
}
