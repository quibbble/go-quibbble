package go_boardgame_networking

import (
	"github.com/gorilla/websocket"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame-networking/pkg/duration"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"time"
)

// GameNetworkOptions are the options required to create a new network
type GameNetworkOptions struct {
	// Games is the list of game builders to add to the networking layer
	Games []bg.BoardGameWithBGNBuilder

	// Adapters allow for external events to be triggered on game start or end
	Adapters []NetworkAdapter

	// GameExpiry refers to how long after creation a game will last before being removed
	GameExpiry time.Duration
}

// CreateGameOptions are the fields necessary for creating a game
type CreateGameOptions struct {
	NetworkOptions *NetworkingCreateGameOptions
	GameOptions    *bg.BoardGameOptions
}

// LoadGameOptions are fields necessary for loading a game
type LoadGameOptions struct {
	NetworkOptions *NetworkingCreateGameOptions
	BGN            *bgn.Game
}

// NetworkingCreateGameOptions are the networking options used to create a game
type NetworkingCreateGameOptions struct {
	// GameKey references the game to play - required
	GameKey string

	// GameID is the unique id assigned to the specific game instance - required
	GameID string

	// Players is a mapping of Team to list of PlayerID - optional
	// If used, all teams passed in BoardGameOptions.Teams must be accounted for in Players
	// This means anyone not in these lists may not join the game
	Players map[string][]string `json:",omitempty"`

	// TurnLengthSeconds refers to the max length of time each player may take per turn - optional
	// nil means no turn length
	TurnLength *duration.Duration `json:",omitempty"`

	// SingleDevice refers to the ability for multiple players to play on one device - optional
	// No business logic is added for this field, used by the frontend only
	SingleDevice bool `json:",omitempty"`
}

// JoinGameOptions are the fields necessary for joining a game
type JoinGameOptions struct {
	GameKey    string
	GameID     string
	PlayerID   string
	PlayerName string
	Conn       *websocket.Conn
}

// OutboundGameMessage is the message sent when returning game information
type OutboundGameMessage struct {
	Type string

	*NetworkingCreateGameOptions

	// Snapshot is the board game data
	Snapshot *bg.BoardGameSnapshot

	// TurnTimeLeft refers to the remaining amount of time in the turn
	TurnTimeLeft string `json:",omitempty"`
}

// OutboundChatMessage is the most recent chat message
type OutboundChatMessage struct {
	Type string

	// Chat the most recent message
	ChatMsg *ChatMessage
}

// OutboundConnectedMessage is the players connected to the game
type OutboundConnectedMessage struct {
	Type string

	// Connected is the list of player names to team
	Connected map[string]string
}

// OutboundErrorMessage is the message sent when there was an error
type OutboundErrorMessage struct {
	Type string

	Error string
}

// ChatMessage is a message in a chat
type ChatMessage struct {
	Name string
	Msg  string
}
