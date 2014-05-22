package procd

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

type RunState int

const (
	RunStateStopped RunState = iota
	RunStateRunning
	RunStatePending
)

func (r RunState) String() string {
	switch r {
	case RunStateStopped:
		return "stopped"
	case RunStateRunning:
		return "running"
	case RunStatePending:
		return "pending"
	default:
		return "unknown"
	}
}

type Process struct {
	m     sync.RWMutex
	wg    *sync.WaitGroup
	pcmd  *exec.Cmd
	state RunState
	cwd   string
	exe   string
	args  []string
}

func NewProcess(cwd, exe string, args []string) *Process {
	return &Process{
		wg:    &sync.WaitGroup{},
		state: RunStateStopped,
		cwd:   cwd,
		exe:   exe,
		args:  args,
	}
}

func (p *Process) State() RunState {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.state
}

// Start starts the process and watches it until it dies
func (p *Process) Start() error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.state != RunStateStopped {
		return fmt.Errorf("Process is not stopped, can't start in state %q", p.state)
	}
	p.state = RunStatePending
	p.pcmd = exec.Command(p.exe, p.args...)
	p.pcmd.Dir = p.cwd
	err := p.pcmd.Start()
	if err != nil {
		log.Printf("Could not start process: %q", err)
		p.state = RunStateStopped
		return err
	}
	p.state = RunStateRunning

	p.wg.Add(1)
	go func(p *Process) {
		defer p.wg.Done()
		err = p.pcmd.Wait()
		p.m.Lock()
		p.state = RunStateStopped
		p.m.Unlock()
		if err != nil {
			log.Printf("Command stopped with error:", err)
		}
	}(p)
	return nil
}

func (p *Process) Stop() error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.state != RunStateRunning {
		return fmt.Errorf("Process is not running, can't stop in state %q", p.state)
	}
	if p.pcmd != nil && p.pcmd.Process != nil {
		if err := p.pcmd.Process.Kill(); err != nil {
			log.Printf("Could not kill process: %q", err)
			return err
		}
	}
	return nil
}

// Wait blocks until process has ended
func (p *Process) Wait() {
	p.wg.Wait()
}
