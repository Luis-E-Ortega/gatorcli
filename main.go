package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		os.Exit(1)
	}

	stateStruct := state{}
	stateStruct.cfg = &data

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
	stateStruct.db = dbQueries
	if stateStruct.db == nil {
		fmt.Println("Error caused by database having a nil pointer")
		os.Exit(1)
	}

	cmds := commands{
		allCommands: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlersLogin)

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

	err = cmds.run(&stateStruct, cmd)
	if err != nil {
		fmt.Printf("Error running command: %v", err)
		os.Exit(1)
	}
}
