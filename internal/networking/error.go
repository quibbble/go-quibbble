package go_boardgame_networking

import (
	"fmt"
	"strings"
)

var (
	ErrBGNUnsupported = func(gameKey string) error {
		return fmt.Errorf("bgn is not supported for gameKey '%s'", gameKey)
	}

	ErrNoExistingGameKey = func(gameKey string) error {
		return fmt.Errorf("gameKey does not exist for gameKey '%s'", gameKey)
	}

	ErrExistingGameID = func(gameKey, gameID string) error {
		return fmt.Errorf("gameID '%s' already exists for gameKey '%s'", gameID, gameKey)
	}

	ErrNoExistingGameID = func(gameKey, gameID string) error {
		return fmt.Errorf("gameID '%s' does not exist for gameKey '%s'", gameID, gameKey)
	}

	ErrCreateGame = func(gameKey, gameID string) error {
		return fmt.Errorf("failed to create game with gameID '%s' for gameKey '%s'", gameID, gameKey)
	}
	ErrStoreGame = func(gameKey, gameID string) error {
		return fmt.Errorf("failed to store game with gameID '%s' for gameKey '%s'", gameID, gameKey)
	}

	ErrHubClosure = func(gameKey ...string) error {
		return fmt.Errorf("game hubs '%s' failed to close gracefully", strings.Join(gameKey, ", "))
	}

	ErrInconsistentTeams = func(gameKey, gameID string) error {
		return fmt.Errorf("number of teams are inconsistent in gameID '%s' for gameKey '%s'", gameID, gameKey)
	}

	ErrCreateGameOptions = func(gameKey, gameID string) error {
		return fmt.Errorf("invalid create game options in gameID '%s' for gameKey '%s'", gameID, gameKey)
	}

	ErrPlayerAlreadyConnected = func(gameKey, gameID string) error {
		return fmt.Errorf("player already connected in gameID '%s' for gameKey '%s'", gameID, gameKey)
	}

	ErrPlayerUnauthorized = func(gameKey, gameID string) error {
		return fmt.Errorf("player not authorized to join gameID '%s' for gameKey '%s'", gameID, gameKey)
	}

	ErrActionNotAllowed = func(actionType string) error {
		return fmt.Errorf("%s action not allowed", actionType)
	}

	ErrNoActionToUndo = fmt.Errorf("no action to undo")

	ErrWrongTeamAction = fmt.Errorf("cannot perform game action for another team")

	ErrInvalidTeam = fmt.Errorf("invalid team")

	ErrAlreadyInTeam = fmt.Errorf("already in team")

	ErrNoOpenTeam = fmt.Errorf("no open team")

	ErrMaxChat = fmt.Errorf("max chat limit reached")
)
