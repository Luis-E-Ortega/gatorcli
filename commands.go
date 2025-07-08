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

func (c *commands) reset(s *state, cmd command) error {
	err := s.db.ResetTables(context.Background())
	if err != nil {
		fmt.Println("Error resetting table")
		return err
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

func handlerAddfeed(s *state, cmd command) error {
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

	userId := user.ID

	feed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      feedName,
			Url:       feedUrl,
			UserID:    userId,
		},
	)
	if err != nil {
		return err
	}
	fmt.Println(feed)

	return nil
}

func (c *command) follow(s *state, cmd command) error {
	url := cmd.arguments[1]
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
