package datastore

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/pkg/logger"
)

type CockroachClient struct {
	pool *pgxpool.Pool
}

func NewCockroachClient(config *CockroachConfig) (*CockroachClient, error) {
	if !config.Enabled {
		return &CockroachClient{}, nil
	}

	cfg, err := pgxpool.ParseConfig(config.GetURL())
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to parse cockroach url")
		return nil, ErrGameStoreConnection
	}

	cfg.MaxConns = 3
	cfg.MinConns = 0
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = time.Minute * 30
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to connect to cockroach")
		return nil, ErrGameStoreConnection
	}

	return &CockroachClient{
		pool: pool,
	}, nil
}

func (c *CockroachClient) GetGame(gameKey, gameID string) (*Game, error) {
	if c.pool == nil {
		return nil, ErrGameStoreNotEnabled
	}

	sql := `
		SELECT bgn, created_at, updated_at, play_count FROM quibbble.games
		WHERE game_key=$1
		AND game_id=$2
	`
	row := c.pool.QueryRow(context.Background(), sql, gameKey, gameID)

	var (
		raw                  string
		createdAt, updatedAt time.Time
		playCount            int
	)

	if err := row.Scan(&raw, &createdAt, &updatedAt, &playCount); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrGameStoreNotFound
		}
		return nil, ErrGameStoreSelect
	}

	logger.Log.Debug().Msgf("found '%s' with id '%s' in game store", gameKey, gameID)

	game, err := bgn.Parse(raw)
	if err != nil {
		return nil, err
	}

	return &Game{
		GameKey:   gameKey,
		GameID:    gameID,
		BGN:       game,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		PlayCount: playCount,
	}, nil
}

func (c *CockroachClient) GetStats(games []string) (*Stats, error) {
	if c.pool == nil {
		return nil, ErrGameStoreNotEnabled
	}

	sql := `
		SELECT game_key, COUNT(game_id) AS games_created, SUM(play_count) AS games_played FROM quibbble.games
		GROUP BY game_key
	`

	rows, err := c.pool.Query(context.Background(), sql)
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to query cockroach")
		if err == pgx.ErrNoRows {
			return nil, ErrGameStoreNotFound
		}
		return nil, ErrGameStoreSelect
	}

	stats := &Stats{
		GamesCreated: make(map[string]int),
		GamesPlayed:  make(map[string]int),
	}

	for _, game := range games {
		stats.GamesCreated[game] = 0
		stats.GamesPlayed[game] = 0
	}

	var (
		gameKey                   string
		gamesCreated, gamesPlayed int
	)

	for rows.Next() {
		if err := rows.Scan(&gameKey, &gamesCreated, &gamesPlayed); err != nil {
			logger.Log.Error().Caller().Err(err).Msg("failed to query cockroach")
			if err == pgx.ErrNoRows {
				return nil, ErrGameStoreNotFound
			}
			return nil, ErrGameStoreSelect
		}
		stats.GamesCreated[gameKey] = gamesCreated
		stats.GamesPlayed[gameKey] = gamesPlayed
	}

	return stats, nil
}

func (c *CockroachClient) Store(game *Game) error {
	if c.pool == nil {
		return ErrGameStoreNotEnabled
	}

	sql := `
		UPSERT INTO quibbble.games (game_key, game_id, bgn, created_at, updated_at, play_count)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := c.pool.Exec(context.Background(), sql, game.GameKey, game.GameID, game.BGN.String(), game.CreatedAt, game.UpdatedAt, game.PlayCount)
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to exec on cockroach")
		return ErrGameStoreInsert
	}

	logger.Log.Debug().Msgf("stored '%s' with id '%s' in game store", game.GameKey, game.GameID)

	return nil
}

func (c *CockroachClient) Close(ctx context.Context) error {
	if c.pool == nil {
		return nil
	}
	c.pool.Close()
	return nil
}
