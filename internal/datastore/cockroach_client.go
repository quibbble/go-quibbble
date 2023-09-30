package datastore

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

type CockroachClient struct {
	conn *pgx.Conn
}

func NewCockroachClient(config *CockroachConfig) (*CockroachClient, error) {
	if !config.Enabled {
		return &CockroachClient{}, nil
	}
	conn, err := pgx.Connect(context.Background(), config.GetURL())
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to connect to cockroach")
		return nil, ErrGameStoreConnection
	}
	return &CockroachClient{
		conn: conn,
	}, nil
}

func (c *CockroachClient) GetGame(gameKey, gameID string) (*Game, error) {
	if c.conn == nil {
		return nil, ErrGameStoreNotEnabled
	}
	sql := `
		SELECT bgn, created_at, updated_at, play_count FROM quibbble.games
		WHERE game_key=$1
		AND game_id=$2
	`
	row := c.conn.QueryRow(context.Background(), sql, gameKey, gameID)

	var (
		bgn                  string
		createdAt, updatedAt time.Time
		playCount            int
	)

	if err := row.Scan(&bgn, &createdAt, &updatedAt, &playCount); err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to query cockroach")
		if err == pgx.ErrNoRows {
			return nil, ErrGameStoreNotFound
		}
		return nil, ErrGameStoreSelect
	}

	logger.Log.Debug().Msgf("found '%s' with id '%s' in game store", gameKey, gameID)

	return &Game{
		GameKey:   gameKey,
		GameID:    gameID,
		BGN:       bgn,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		PlayCount: playCount,
	}, nil
}

func (c *CockroachClient) GetStats() (*Stats, error) {
	// TODO
	return &Stats{
		GamesPlayed:    make(map[string]int),
		GamesCompleted: make(map[string]int),
	}, nil
}

func (c *CockroachClient) Store(game *Game) error {
	if c.conn == nil {
		return ErrGameStoreNotEnabled
	}
	sql := `
		UPSERT INTO quibbble.games (game_key, game_id, bgn, created_at, updated_at, play_count)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := c.conn.Exec(context.Background(), sql, game.GameKey, game.GameID, game.BGN, game.CreatedAt, game.UpdatedAt, game.PlayCount)
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to exec on cockroach")
		return ErrGameStoreInsert
	}
	return nil
}

func (c *CockroachClient) Close(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close(ctx)
}
