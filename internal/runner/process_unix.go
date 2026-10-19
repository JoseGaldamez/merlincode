//go:build !windows

package runner

import (
	"errors"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type unixProcess struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	done bool
}

func startProcess(executable string, args []string, directory string, env []string, output io.Writer) (childProcess, error) {
	cmd := exec.Command(executable, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = directory, env, output, output
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &unixProcess{cmd: cmd}, nil
}

func (p *unixProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return nil
	}
	err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func (p *unixProcess) Wait() (int, error) {
	err := p.cmd.Wait()
	p.mu.Lock()
	// Keep children from surviving a launcher that exits before its descendants.
	killErr := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	p.done = true
	p.mu.Unlock()
	if errors.Is(killErr, syscall.ESRCH) {
		killErr = nil
	}
	return p.cmd.ProcessState.ExitCode(), errors.Join(err, killErr)
}
