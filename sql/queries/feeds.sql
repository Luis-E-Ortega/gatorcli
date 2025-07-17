-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
$1,
$2,
$3,
$4,
$5,
$6
)
RETURNING *;

-- name: GetFeeds :many
SELECT feeds.name, url, users.name AS username
FROM feeds
INNER JOIN users ON users.id = feeds.user_id;

-- name: CreateFeedFollow :many
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
    )
    RETURNING *
)

SELECT
    inserted_feed_follow.*, 
    feeds.name AS feed_name, 
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN users ON users.id = inserted_feed_follow.user_id
INNER JOIN feeds ON feeds.id = inserted_feed_follow.feed_id;

-- name: GetFeedByURL :one
SELECT *
FROM feeds
WHERE feeds.url = $1;

-- name: GetFeedFollowsForUser :many
SELECT 
    *, 
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
INNER JOIN users ON users.id = feed_follows.user_id
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE users.id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE feed_follows.user_id = $1 AND feed_follows.feed_id = (SELECT id from feeds WHERE url = $2);

-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = $1, updated_at = $2
WHERE id = $3;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;