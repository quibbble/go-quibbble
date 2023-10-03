package go_boardgame_networking

import (
	"time"

	"github.com/gorilla/websocket"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/internal/datastore"
	"github.com/quibbble/go-quibbble/pkg/duration"
)

// GameNetworkOptions are the options required to create a new network
type GameNetworkOptions struct {
	// Games is the list of game builders to add to the networking layer
	Games []bg.BoardGameBuilder

	// Adapters allow for external events to be triggered on game start or end
	Adapters []NetworkAdapter

	// GameExpiry refers to how long after creation a game will last before being removed
	GameExpiry time.Duration

	// GameStore stores games longterm
	GameStore datastore.GameStore
}

// CreateGameOptions are the fields necessary for creating a game
type CreateGameOptions struct {
	NetworkOptions *NetworkingCreateGameOptions

	// One of the following is required in order to create a game
	GameOptions *bg.BoardGameOptions
	BGN         *bgn.Game
	GameData    *datastore.Game
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

	// TurnLength refers to the max length of time each player may take per turn - optional
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

// OutboundMessage is the message sent to player
type OutboundMessage struct {
	Type string

	Payload interface{}
}

type outboundNetworkMessage struct {
	*NetworkingCreateGameOptions

	// Name is the name of the player receiving the message
	Name string

	// TurnTimeLeft refers to the remaining amount of time in the turn
	TurnTimeLeft string `json:",omitempty"`
}

// ChatMessage is a message in a chat
type ChatMessage struct {
	Name string
	Msg  string
}
