package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

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

func handlerLogin(s *state, cmd command) error {
	// First check to ensure arguments isn't empty
	if len(cmd.arguments) < 1 {
		err := errors.New("username required")
		return err
	}

	// Get user and check error to make sure a user exists before allowing login
	_, err := s.db.GetUser(context.Background(), cmd.arguments[0])
	if err == sql.ErrNoRows {
		fmt.Printf("Username does not exist!\n")
		os.Exit(1)
	}

	err = s.cfg.SetUser(cmd.arguments[0])
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
			fmt.Printf("Error running run method: %v", err)
			return err
		}
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.allCommands[name] = f
}
