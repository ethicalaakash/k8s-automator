package os

import (
	"os/exec"
)

// Commander defines an interface for executing shell commands
type Commander interface {
	// Execute runs a command and returns its combined output
	Execute(name string, args ...string) ([]byte, error)
}

// RealCommander implements Commander using the actual os/exec package
type RealCommander struct{}

// Execute runs a command and returns its combined output
func (c *RealCommander) Execute(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// NewCommander creates a real command executor
func NewCommander() Commander {
	return &RealCommander{}
}
