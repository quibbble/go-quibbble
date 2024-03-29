package go_boardgame_networking

import bg "github.com/quibbble/go-boardgame"

// NetworkAdapter allows for external events to be triggered on game start and end if so desired
// This can be useful for updating statistics, storing completed games, etc.
type NetworkAdapter interface {
	OnGameStart(initialOptions *CreateGameOptions)
	// OnGameUpdate
	OnGameEnd(snapshot *bg.BoardGameSnapshot, options *NetworkingCreateGameOptions)
	// OnGameClean
}
