package main

import (
	"encoding/json"
	"fmt"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
)

func main() {
	data, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		return
	}

	err = data.SetUser("Luis")
	if err != nil {
		fmt.Println("Error setting user")
		return
	}

	updatedData, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		return
	}

	printableData, err := json.Marshal(updatedData)
	if err != nil {
		fmt.Println("Error marshalling file")
		return
	}

	stringVersion := string(printableData)

	fmt.Println(stringVersion)

}
