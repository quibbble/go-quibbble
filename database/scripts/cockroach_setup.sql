-- create the quibbble schema
CREATE SCHEMA IF NOT EXISTS quibbble;

-- create the games table
CREATE TABLE IF NOT EXISTS quibbble.games (
    game_key STRING NOT NULL,
    game_id STRING NOT NULL,
    bgn STRING,
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	play_count INT,
    CONSTRAINT id PRIMARY KEY (game_key, game_id)
);

-- get a game from the games table
SELECT * FROM quibbble.games
WHERE game_key=$1
AND game_id=$2

-- insert a game into the games table
INSERT INTO quibbble.games (game_key, game_id, bgn, created_at, updated_at, play_count)
VALUES ($1, $2, $3, $4, $5, $6)
