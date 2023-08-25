package adapters

import (
	"fmt"
	"strings"

	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-quibbble/internal/datastore"
	networking "github.com/quibbble/go-quibbble/internal/networking"
	"github.com/rs/zerolog"
)

const ignoreGameID = "test"

type RedisAdapter struct {
	client *datastore.RedisClient
	log    zerolog.Logger
}

func NewRedisAdapter(redis *datastore.RedisClient, log zerolog.Logger) *RedisAdapter {
	return &RedisAdapter{
		client: redis,
		log:    log,
	}
}

func (r *RedisAdapter) OnGameStart(initialOptions *networking.CreateGameOptions) {
	if strings.EqualFold(initialOptions.NetworkOptions.GameID, ignoreGameID) {
		return
	}
	key := fmt.Sprintf(datastore.StatsKeyGamesPlayed, strings.ToLower(initialOptions.NetworkOptions.GameKey))
	if err := r.client.IncrStat(key); err != nil {
		r.log.Debug().Caller().Err(err).Msgf("failed to incr games played for key %s", key)
	}
}

func (r *RedisAdapter) OnGameEnd(snapshot *bg.BoardGameSnapshot, options *networking.NetworkingCreateGameOptions) {
	if strings.EqualFold(options.GameID, ignoreGameID) {
		return
	}
	key := fmt.Sprintf(datastore.StatsKeyGamesCompleted, strings.ToLower(options.GameKey))
	if err := r.client.IncrStat(key); err != nil {
		r.log.Debug().Caller().Err(err).Msgf("failed to incr games played for key %s", key)
	}
}
