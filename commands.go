package main

import (
	"errors"
	"fmt"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
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

	err := s.cfg.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Println("User has been successfully set")
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	if inputCommand, ok := c.allCommands[cmd.name]; ok {
		err := inputCommand(s, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.allCommands[name] = f
}
