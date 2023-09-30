package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func NewLogger(cfg Config, environment string) (zerolog.Logger, error) {
	lvl, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return zerolog.Logger{}, err
	}
	logger := zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
	if environment == "local" {
		logger = logger.Output(zerolog.ConsoleWriter{
			Out: os.Stderr,
		})
	}
	return logger, nil
}
