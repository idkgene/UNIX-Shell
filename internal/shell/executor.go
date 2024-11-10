package shell

import (
	"context"
	"io"
	"os/exec"
	"syscall"
	"fmt"
	"sync"
)

type Executor struct {
    shell       *Shell
    mu          sync.Mutex
}

type ProcessGroup struct {
    Commands        []*exec.Cmd
    Pgid            int
    Cancel          context.CancelFunc
}

func NewExecutor(shell *Shell) *Executor {
    return &Executor{
        shell: shell,
    }
}

func (e *Executor) Execute(ctx context.Context, pipeline []Command) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    if len(pipeline) == 0 {
        return nil
    }

    pg := &ProcessGroup{
        Commands: make([]*exec.Cmd, len(pipeline)),
    }

    ctx, cancel := context.WithCancel(ctx)
    pg.Cancel = cancel

    for i, cmd := range pipeline {
        execCmd := e.prepareCommand(ctx, cmd)
        pg.Commands[i] = execCmd
    }

    if err := e.setupPipes(pg.Commands); err != nil {
        return err
    }

    if err := e.startCommands(pg); err != nil {
        pg.Cancel()
        return err
    }

    e.shell.processGroup.Store(pg.Pgid, pg)

    return e.waitCommands(pg)
}

func (e *Executor) startCommands(pg *ProcessGroup) error {
	for i, cmd := range pg.Commands {
			if i == 0 {
					cmd.SysProcAttr = &syscall.SysProcAttr{
						Setpgid: true,
					}
			}

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start command: %w", err)
			}

			if i == 0 {
			    pg.Pgid = cmd.Process.Pid
			}
	}
	
    return nil
}

func (e *Executor) waitCommands(pg *ProcessGroup) error {
	for _, cmd := range pg.Commands {
			if err := cmd.Wait(); err != nil {
					return err
			}
	}
	
    return nil
}

func (e *Executor) prepareCommand(ctx context.Context, cmd Command) *exec.Cmd {
    execCmd := exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
    
    execCmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }

    if cmd.Env != nil {
        execCmd.Env = cmd.Env
    }

    if cmd.Dir != "" {
        execCmd.Dir = cmd.Dir
    }

    return execCmd
}

func (e *Executor) setupPipes(cmds []*exec.Cmd) error {
    for i := 0; i < len(cmds) - 1; i++ {
        r, w := io.Pipe()
        cmds[i].Stdout = w
        cmds[i+1].Stdin = r
    }
    
    return nil
}
