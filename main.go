package main

import (
	"fmt"
	"os"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		os.Exit(1)
	}

	stateStruct := state{}
	stateStruct.config = &data

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
