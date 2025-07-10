package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

type state struct {
	db    *database.Queries
	cfg   *config.Config
	RawDB *sql.DB
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

func (c *commands) reset(s *state, cmd command) error {
	// Access the raw *sql.DB connection directly from the state
	dbConnection := s.RawDB

	if dbConnection == nil {
		return fmt.Errorf("raw database connection in state is nil, cannot run migrations")
	}

	// Save the path for migrations
	const migrationsDir = "./sql/schema"

	goose.SetDialect("postgres")

	fmt.Println("Running database migrations DOWN...")
	err := goose.Down(dbConnection, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to run goose down migrations : %w", err)
	}

	fmt.Println("Running database migrations UP...")
	err = goose.Up(dbConnection, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to run goose up migrations: %w", err)
	}

	fmt.Println("Reset successful!")
	return nil
}

func (c *commands) users(s *state, cmd command) error {
	usersList, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range usersList {
		if user == s.cfg.CurrentUserName {
			fmt.Printf("*%s (current)\n", user)
		} else {
			fmt.Println("*" + user)
		}
	}
	return nil
}

func (c *commands) agg(s *state, cmd command) error {
	rssFeed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(rssFeed, "", " ")
	fmt.Println(string(data))

	return nil
}

func (c *commands) feeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("error occured while trying to pull feeds")
		return err
	}

	if len(feeds) != 0 {
		for _, row := range feeds {
			fmt.Printf("Feed Information: \n Name: %v\n URL: %v\n Username: %v", row.Name, row.Url, row.Username)
		}
	}
	return nil
}

func (c *commands) handlerAddfeed(s *state, cmd command) error {
	// First check to ensure arguments isn't empty
	if len(cmd.arguments) < 2 {
		err := errors.New("name and url required")
		return err
	}

	// Get user input to fill out name and url for the feed
	userInput := cmd.arguments
	feedName := userInput[0]
	feedUrl := userInput[1]

	// Retrieve current user and ensure they are registered/logged in
	currentUser := s.cfg.CurrentUserName

	user, err := s.db.GetUser(context.Background(), currentUser)
	if err != nil {
		return err
	}

	feed, err := s.db.GetFeedByURL(context.Background(), feedUrl)
	if err != nil {
		// Check if the error is specifically 'sql.ErrNoRows'
		if errors.Is(err, sql.ErrNoRows) {
			// Case 1: Feed does NOT exist
			// Proceed to create new feed
			fmt.Println("Feed not found. Creating new feed...")

			feedParams := database.CreateFeedParams{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Name:      feedName,
				Url:       feedUrl,
				UserID:    user.ID,
			}

			newFeed, createErr := s.db.CreateFeed(context.Background(), feedParams)
			if createErr != nil {
				return fmt.Errorf("failed to create new feed: %w", createErr)
			}
			feed = newFeed // Update 'feed' to hold the newly created feed
		} else {
			// Case 2: Some other error occurred while trying to get the new feed
			return fmt.Errorf("error checking for existing feed : %w", err)
		}
	}
	// Case 3: Feed WAS found (err is nil here)
	// Proceed to create the feed follow record using 'feed.ID'

	fmt.Printf("Feed '%s' found/created. Proceeding to follow.\n", feed.Name)

	// Creating a "fake" command to have the proper format to call follow from here
	fakeCmd := command{
		arguments: []string{feedUrl},
	}

	err = c.follow(s, fakeCmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *commands) follow(s *state, cmd command) error {
	// Get user input for url
	// Have to check if this is the correct index
	url := cmd.arguments[0]
	currentUser := s.cfg.CurrentUserName
	user, err := s.db.GetUser(context.Background(), currentUser)
	if err != nil {
		return err
	}

	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return err
	}

	feedsFollowRow, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}

	if len(feedsFollowRow) == 0 {
		return errors.New("failed to create feed follow")
	}
	// Using indexing due to query being marked as :many
	fmt.Println(feedsFollowRow[0].FeedName, feedsFollowRow[0].UserName)

	return nil
}

func (c *commands) following(s *state, cmd command) error {
	currentUser := s.cfg.CurrentUserName
	user, err := s.db.GetUser(context.Background(), currentUser)
	if err != nil {
		return err
	}

	userID := user.ID
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), userID)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		fmt.Println(feed.FeedName)
	}

	return nil
}
