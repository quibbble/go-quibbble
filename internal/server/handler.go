package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	bg "github.com/quibbble/go-boardgame"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/quibbble/go-quibbble/internal/datastore"
	networking "github.com/quibbble/go-quibbble/internal/networking"
	"github.com/quibbble/go-quibbble/pkg/logger"
	"github.com/unrolled/render"
)

type Handler struct {
	render    *render.Render
	network   *networking.GameNetwork
	gameStore datastore.GameStore
}

func NewHandler(render *render.Render, network *networking.GameNetwork, gameStore datastore.GameStore) *Handler {
	return &Handler{
		render:    render,
		network:   network,
		gameStore: gameStore,
	}
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var create CreateGameRequest
	if err := unmarshalJSONRequestBody(r, &create); err != nil {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: err.Error()})
		return
	}
	if create.Teams > len(teams) {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: "too many teams"})
		return
	}
	t := make([]string, 0)
	for i := 0; i < create.Teams; i++ {
		t = append(t, teams[i])
	}
	if err := h.network.CreateGame(networking.CreateGameOptions{
		NetworkOptions: create.NetworkingCreateGameOptions,
		GameOptions: &bg.BoardGameOptions{
			Teams:       t,
			MoreOptions: create.MoreOptions,
		},
	}); err != nil {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) LoadGame(w http.ResponseWriter, r *http.Request) {
	var load LoadGameRequest
	if err := unmarshalJSONRequestBody(r, &load); err != nil {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: err.Error()})
		return
	}
	game, err := bgn.Parse(load.BGN)
	if err != nil {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: err.Error()})
		return
	}
	if game.Tags["Game"] != load.GameKey {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: "game key does not match bgn key"})
		return
	}
	if len(strings.Split(game.Tags["Teams"], ", ")) > len(teams) {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: "too many teams"})
		return
	}
	t := make([]string, 0)
	for i := range strings.Split(game.Tags["Teams"], ", ") {
		t = append(t, teams[i])
	}
	game.Tags["Teams"] = strings.Join(t, ", ")
	if err := h.network.CreateGame(networking.CreateGameOptions{
		NetworkOptions: load.NetworkingCreateGameOptions,
		BGN:            game,
	}); err != nil {
		writeJSONResponse(h.render, w, http.StatusBadRequest, errorResponse{Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) JoinGame(w http.ResponseWriter, r *http.Request) {
	gameKey := r.URL.Query().Get("GameKey")
	gameID := r.URL.Query().Get("GameID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		writeJSONResponse(h.render, w, http.StatusInternalServerError, errorResponse{Message: "failed to upgrade websocket connection"})
		return
	}
	if err := h.network.JoinGame(networking.JoinGameOptions{
		GameKey:    gameKey,
		GameID:     gameID,
		PlayerName: generateName(),
		Conn:       conn,
	}); err != nil {
		_ = conn.Close()
	}
}

func (h *Handler) JoinSecureGame(w http.ResponseWriter, r *http.Request) {
	gameKey := r.URL.Query().Get("GameKey")
	gameID := r.URL.Query().Get("GameID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		writeJSONResponse(h.render, w, http.StatusInternalServerError, errorResponse{Message: "failed to upgrade websocket connection"})
		return
	}
	/*
		TODO------------------------------------------------------------------------------------------------------
		Add some form of authentication here such as jwt tokens that sets player field based on jwt subject field.
		This then creates a secure connection between the game and players such that the players playing the game
		are known to the game and one player cannot make plays for the other.
		TODO------------------------------------------------------------------------------------------------------
	*/
	playerID := ""
	playerName := "" // there should be some lookup to get a player name from player ID
	if err := h.network.JoinGame(networking.JoinGameOptions{
		GameKey:    gameKey,
		GameID:     gameID,
		PlayerID:   playerID,
		PlayerName: playerName,
		Conn:       conn,
	}); err != nil {
		_ = conn.Close()
	}
}

func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	gameKey := r.URL.Query().Get("GameKey")
	gameID := r.URL.Query().Get("GameID")
	team := r.URL.Query().Get("Team")

	var snapshot interface{}
	var err error
	if team != "" {
		snapshot, err = h.network.GetSnapshot(gameKey, gameID, team)
	} else {
		snapshot, err = h.network.GetSnapshot(gameKey, gameID)
	}
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msgf("failed to get game snapshot '%s' for '%s'", gameID, gameKey)
		writeJSONResponse(h.render, w, http.StatusNotFound, errorResponse{Message: err.Error()})
		return
	}
	writeJSONResponse(h.render, w, http.StatusOK, snapshot)
}

func (h *Handler) GetBGN(w http.ResponseWriter, r *http.Request) {
	gameKey := r.URL.Query().Get("GameKey")
	gameID := r.URL.Query().Get("GameID")
	game, err := h.network.GetBGN(gameKey, gameID)
	if err != nil {
		writeJSONResponse(h.render, w, http.StatusNotFound, errorResponse{Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(game.String()))
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	statsStored, err := h.gameStore.GetStats(h.network.GetGames())
	if err != nil {
		logger.Log.Error().Caller().Err(err).Msg("failed to retrieve game stats")
		statsStored = &datastore.Stats{}
	}
	statsCurrent := h.network.GetStats()
	writeJSONResponse(h.render, w, http.StatusOK, StatsResponse{
		GamesCreated:  statsStored.GamesCreated,
		GamesPlayed:   statsStored.GamesPlayed,
		ActiveGames:   statsCurrent.ActiveGames,
		ActivePlayers: statsCurrent.ActivePlayers,
	})
}

func (h *Handler) GetActiveGameIDs(w http.ResponseWriter, r *http.Request) {
	activeGameIDs := h.network.GetActiveGameIDs()
	writeJSONResponse(h.render, w, http.StatusOK, activeGameIDs)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
}

type errorResponse struct {
	Message string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
