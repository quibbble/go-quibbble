package go_boardgame_networking

import bg "github.com/quibbble/go-boardgame"

// NetworkAdapter allows for external events to be triggered on game start and end if so desired
// This can be useful for updating statistics, storing completed games, etc.
// TODO - update this to work with an OnGameUpdate func that is called anytime there is a change in game state
type NetworkAdapter interface {
	OnGameStart(initialOptions *CreateGameOptions)
	OnGameEnd(snapshot *bg.BoardGameSnapshot, options *NetworkingCreateGameOptions)
}
