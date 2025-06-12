package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		os.Exit(1)
	}

	currentState := state{}
	currentState.cfg = &data

	db, err := sql.Open("postgres", data.DbUrl)
	if err != nil {
		fmt.Println("Error opening database channel")
		os.Exit(1)
	}
	err = db.Ping()
	if err != nil {
		fmt.Printf("Error caused by failed ping: %v", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)
	currentState.db = dbQueries
	if currentState.db == nil {
		fmt.Println("Error caused by database having a nil pointer")
		os.Exit(1)
	}

	cmds := commands{
		allCommands: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", cmds.reset)
	cmds.register("users", cmds.users)

	userInput := os.Args
	if len(userInput) < 2 {
		println("not enough arguments")
		os.Exit(1)
	}

	cmdName := userInput[1]
	cmdArgs := userInput[2:]

	cmd := command{
		name:      cmdName,
		arguments: cmdArgs,
	}

	err = cmds.run(&currentState, cmd)
	if err != nil {
		fmt.Printf("Error running command: %v", err)
		os.Exit(1)
	}
}

// Moved to here from commands because it was not working there
func handlerRegister(s *state, cmd command) error {
	// Check to ensure name isn't empty
	if len(cmd.arguments) < 1 {
		return errors.New("name required")
	}

	params := database.CreateUserParams{}
	params.Name = cmd.arguments[0]
	params.ID = uuid.New()
	params.CreatedAt = time.Now()
	params.UpdatedAt = time.Now()

	user, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			os.Exit(1)
		}
		fmt.Printf("ERROR WITH HANDLER STOPPING ON CHECK %v", err)
		return err
	}

	s.cfg.SetUser(user.Name)
	fmt.Println("New user created!")
	log.Printf("New user logged: %+v", user)

	return nil
}
