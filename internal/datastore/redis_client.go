package datastore

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
)

const (
	StatsKeyGamesPlayed    = "stats:%s:games_played"    // EX: "stats:tsuro:games_played"
	StatsKeyGamesCompleted = "stats:%s:games_completed" // EX: "stats:stratego:games_completed"

	// GameStorageKey = "game:%s:%s" // EX: "game:connect4:excited-parrot"
)

var errRedisNotEnabled = fmt.Errorf("redis is not enabled")

type RedisClient struct {
	client *redis.Client
	config *RedisConfig
}

func NewRedisClient(config *RedisConfig) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     config.Host,
			Password: config.Password,
			DB:       0,
		}),
		config: config,
	}
}

type GameStatsAllTime struct {
	GamesPlayed    map[string]int
	GamesCompleted map[string]int
}

func (r *RedisClient) IncrStat(key string) error {
	if !r.config.Enabled {
		return errRedisNotEnabled
	}
	_, err := r.client.Incr(key).Result()
	return err
}

func (r *RedisClient) GetGameStats(games []string) (*GameStatsAllTime, error) {
	if !r.config.Enabled {
		return nil, errRedisNotEnabled
	}
	stats := &GameStatsAllTime{
		GamesPlayed:    make(map[string]int),
		GamesCompleted: make(map[string]int),
	}
	for _, game := range games {
		gamesPlayedStr, err := r.client.Get(fmt.Sprintf(StatsKeyGamesPlayed, strings.ToLower(game))).Result()
		if err != nil {
			continue
		}
		gamesCompletedStr, err := r.client.Get(fmt.Sprintf(StatsKeyGamesCompleted, strings.ToLower(game))).Result()
		if err != nil {
			continue
		}
		gamesPlayed, err := strconv.Atoi(gamesPlayedStr)
		if err != nil {
			return nil, err
		}
		gamesCompleted, err := strconv.Atoi(gamesCompletedStr)
		if err != nil {
			return nil, err
		}
		stats.GamesPlayed[game] = gamesPlayed
		stats.GamesCompleted[game] = gamesCompleted
	}
	return stats, nil
}
