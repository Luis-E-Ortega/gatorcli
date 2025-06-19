package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func main() {
	// Read file and save to variable
	data, err := config.Read()
	if err != nil {
		fmt.Println("Error reading file")
		os.Exit(1)
	}

	currentState := state{}
	currentState.cfg = &data

	// Open the channel to the database
	db, err := sql.Open("postgres", data.DbUrl)
	if err != nil {
		fmt.Println("Error opening database channel")
		os.Exit(1)
	}
	err = db.Ping() // Ping check to ensure connection is active
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
	cmds.register("agg", cmds.agg)
	cmds.register("addfeed", handlerAddfeed)

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

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	rssFeed := RSSFeed{}

	// Make a request using this method for more control to set headers
	newReq, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	// Set the specific header to our project name
	newReq.Header.Set("User-Agent", "gator")

	// Create client
	client := http.Client{}

	resp, err := client.Do(newReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(body, &rssFeed)
	if err != nil {
		return nil, err
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)

	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}

	return &rssFeed, nil
}
