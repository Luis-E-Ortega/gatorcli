package main

import (
	"errors"
	"fmt"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
)

type state struct {
	config *config.Config
}

type command struct {
	name      string
	arguments []string
}

type commands struct {
	allCommands map[string]func(*state, command) error
}

func handlersLogin(s *state, cmd command) error {
	// First check to ensure arguments isn't empty
	if len(cmd.arguments) < 1 {
		err := errors.New("username required")
		return err
	}

	err := s.config.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Println("User has been successfully set")
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {

}
