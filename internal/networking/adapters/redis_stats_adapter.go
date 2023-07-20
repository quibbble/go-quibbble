package adapters

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis"
	bg "github.com/quibbble/go-boardgame"
	networking "github.com/quibbble/go-quibbble/internal/networking"
	"github.com/rs/zerolog"
)

const (
	gamesPlayedStatsKey    = "stats:%s:games_played"    // EX: "stats:tsuro:games_played"
	gamesCompletedStatsKey = "stats:%s:games_completed" // EX: "stats:stratego:games_completed"

	// gameStorageKey = "game:%s:%s" // EX: "game:connect4:excited-parrot"
)

type RedisAdapterConfig struct {
	Enabled      bool
	Host         string
	Password     string
	IgnoreGameID string
}

type RedisAdapter struct {
	config *RedisAdapterConfig
	client *redis.Client
	log    zerolog.Logger
}

func NewRedisAdapter(config *RedisAdapterConfig, log zerolog.Logger) RedisAdapter {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Host,
		Password: config.Password,
		DB:       0,
	})
	return RedisAdapter{
		config: config,
		client: client,
		log:    log,
	}
}

func (r *RedisAdapter) OnGameStart(initialOptions *networking.CreateGameOptions) {
	if !r.config.Enabled || strings.EqualFold(initialOptions.NetworkOptions.GameID, r.config.IgnoreGameID) {
		return
	}
	key := fmt.Sprintf(gamesPlayedStatsKey, strings.ToLower(initialOptions.NetworkOptions.GameKey))
	_, err := r.client.Incr(key).Result()
	if err != nil {
		r.log.Error().Caller().Err(err).Msgf("failed to incr games played for key %s", key)
	}
}

func (r *RedisAdapter) OnGameEnd(snapshot *bg.BoardGameSnapshot, options *networking.NetworkingCreateGameOptions) {
	if !r.config.Enabled || strings.EqualFold(options.GameID, r.config.IgnoreGameID) {
		return
	}
	key := fmt.Sprintf(gamesCompletedStatsKey, strings.ToLower(options.GameKey))
	_, err := r.client.Incr(key).Result()
	if err != nil {
		r.log.Error().Caller().Err(err).Msgf("failed to incr games completed for key %s", key)
	}
}
