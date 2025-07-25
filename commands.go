package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Luis-E-Ortega/gatorcli/internal/config"
	"github.com/Luis-E-Ortega/gatorcli/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

// Used to login as a specific user
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

// Dispatches the requested CLI command by looking up its handler and executing it
func (c *commands) run(s *state, cmd command) error {
	if inputCommand, ok := c.allCommands[cmd.name]; ok {
		err := inputCommand(s, cmd)
		if err != nil {
			fmt.Printf("Error running run method: %v", err)
			return err
		}
	} else {
		fmt.Printf("Unknown command : %s\n", cmd.name)
		return fmt.Errorf("unkown command: %s", cmd.name)
	}
	return nil
}

// Used to register a new user
func (c *commands) register(name string, f func(*state, command) error) {
	c.allCommands[name] = f
}

// Resets the database, running goose migrations down and up
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

// Displays a list of all registered users
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

// Continuously running program to check for (and apply) updates to feeds at a given interval
func (c *commands) agg(s *state, cmd command) error {
	if len(cmd.arguments) < 1 {
		return errors.New("missing time_between_reqs argument")
	}
	time_between_reqs, err := time.ParseDuration(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %v\n", time_between_reqs)
	ticker := time.NewTicker(time_between_reqs)
	defer ticker.Stop()

	for {
		// Call scrapeFeeds function
		err := c.scrapeFeeds(s)
		if err != nil {
			fmt.Println("Error scraping feeds:", err)
		}

		<-ticker.C // wait for next tick
	}
}

// Used by agg to fetch feeds and keep database updated while running
func (c *commands) scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	now := time.Now()
	err = s.db.MarkFeedFetched(
		context.Background(),
		database.MarkFeedFetchedParams{
			LastFetchedAt: sql.NullTime{Time: now, Valid: true},
			UpdatedAt:     now,
			ID:            nextFeed.ID,
		},
	)
	if err != nil {
		return err
	}

	parsedFeed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}

	for _, item := range parsedFeed.Channel.Item {
		pubDateStr := item.PubDate
		layout := time.RFC1123

		// Format publish date to match the variable type in params
		publishedAt, err := time.Parse(layout, pubDateStr)
		if err != nil {
			return err
		}
		// Format description to match the variable type in params
		desc := sql.NullString{
			String: item.Description,
			Valid:  item.Description != "",
		}
		_, err = s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       item.Title,
				Url:         item.Link,
				Description: desc,
				PublishedAt: publishedAt,
				FeedID:      nextFeed.ID,
			})
		if err != nil {
			// This error comes from the pq package(Postgres driver)
			if pgErr, ok := err.(*pq.Error); ok {
				if pgErr.Code == "23505" { // unique_violation error code
					// Means there was a duplicate URL, simply skip
					continue
				}
			}
			// For other errors, return or handle
			log.Printf("failed to create post: %v", err)
			return err
		}
	}

	return nil
}

// Displays info on followed posts, optional limit for how many to display at once
func (c *commands) browse(s *state, cmd command) error {
	limit := 2 // Set default limit
	if len(cmd.arguments) > 0 {
		if parsed, err := strconv.Atoi(cmd.arguments[0]); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	currentUser := s.cfg.CurrentUserName
	if currentUser == "" {
		return errors.New("must be logged in to browse posts")
	}
	user, err := s.db.GetUser(context.Background(), currentUser)
	if err != nil {
		return err
	}
	postsList, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			UserID: user.ID,
			Limit:  int32(limit),
		})
	if err != nil {
		return err
	}
	for _, post := range postsList {
		fmt.Printf("Title: %s\nURL: %s\nPublished: %s\n\n", post.Title, post.Url, post.PublishedAt.Format(time.RFC1123))
	}
	return nil
}

// Shows all of the feeds information across users
func (c *commands) feeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("error occured while trying to pull feeds")
		return err
	}

	if len(feeds) != 0 {
		for _, row := range feeds {
			fmt.Printf("Feed Information: \n Name: %v\n URL: %v\n Username: %v\n", row.Name, row.Url, row.Username)
		}
	}
	return nil
}

// Used to add a feed to our program that users can then follow after
// (automatically follows the feed  on command run for the current logged in user)
func (c *commands) handlerAddfeed(s *state, cmd command, user database.User) error {
	// First check to ensure arguments isn't empty
	if len(cmd.arguments) < 2 {
		err := errors.New("name and url required")
		return err
	}

	// Get user input to fill out name and url for the feed
	userInput := cmd.arguments
	feedName := userInput[0]
	feedUrl := userInput[1]

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

	err = c.follow(s, fakeCmd, user)
	if err != nil {
		return err
	}

	return nil
}

// Follows a feed specifically for the logged in user
func (c *commands) follow(s *state, cmd command, user database.User) error {
	// Get user input for url
	url := cmd.arguments[0]

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

// Used to unfollow a feed
func (c *commands) unfollow(s *state, cmd command, user database.User) error {
	if len(cmd.arguments) < 1 {
		return errors.New("not enough arguments passed into command")
	}
	feedURL := cmd.arguments[0]
	err := s.db.DeleteFeedFollow(
		context.Background(),
		database.DeleteFeedFollowParams{
			UserID: user.ID,
			Url:    feedURL,
		})

	if err != nil {
		return err
	}
	fmt.Println("Feed successfully unfollowed")
	return nil
}

// Shows list of all feeds followed by user
func (c *commands) following(s *state, cmd command, user database.User) error {
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

// Wrapper function used to authenticate login information
func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		currentUser := s.cfg.CurrentUserName
		if currentUser == "" {
			return fmt.Errorf("no user logged in")
		}
		user, err := s.db.GetUser(context.Background(), currentUser)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}
