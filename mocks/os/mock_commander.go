package mock_os

import (
	osCmd "github.com/ethicalaakash/k8s-automator/internal/os"
)

// MockCommander implements Commander for testing
type MockCommander struct {
	// Map of commands to their responses
	Responses map[string]MockResponse
	// Handlers for dynamic command responses
	Handlers map[string]func(string, ...string) ([]byte, error)
	// Function call tracking
	ExecutedCommands []string
}

// MockResponse holds mock output for a command
type MockResponse struct {
	Output []byte
	Error  error
}

// NewMockCommander creates a mock command executor
func NewMockCommander() *MockCommander {
	return &MockCommander{
		Responses:        make(map[string]MockResponse),
		Handlers:         make(map[string]func(string, ...string) ([]byte, error)),
		ExecutedCommands: make([]string, 0),
	}
}

// AddResponse adds a predefined response for a command
func (m *MockCommander) AddResponse(command string, output []byte, err error) {
	m.Responses[command] = MockResponse{
		Output: output,
		Error:  err,
	}
}

// AddHandler adds a dynamic handler for a command
func (m *MockCommander) AddHandler(commandPrefix string, handler func(string, ...string) ([]byte, error)) {
	m.Handlers[commandPrefix] = handler
}

// Execute runs a mocked command and returns predetermined output
func (m *MockCommander) Execute(name string, args ...string) ([]byte, error) {
	// Build the command string
	command := name
	if len(args) > 0 {
		command += " " + args[0]
	}
	fullCommand := name
	for _, arg := range args {
		fullCommand += " " + arg
	}

	// Track the executed command
	m.ExecutedCommands = append(m.ExecutedCommands, fullCommand)

	// Check for a handler first
	if handler, exists := m.Handlers[command]; exists {
		return handler(name, args...)
	}

	// Check for a predefined response
	if response, exists := m.Responses[fullCommand]; exists {
		return response.Output, response.Error
	}

	// Return a default error for unmocked commands
	return nil, &MockCommandError{Command: fullCommand, Message: "command not mocked"}
}

// MockCommandError is an error type for mock command errors
type MockCommandError struct {
	Command string
	Message string
}

func (e *MockCommandError) Error() string {
	return e.Message + ": " + e.Command
}

// Ensure MockCommander implements the Commander interface
var _ osCmd.Commander = (*MockCommander)(nil)
