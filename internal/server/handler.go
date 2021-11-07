package server

import (
	"github.com/gorilla/websocket"
	bg "github.com/quibbble/go-boardgame"
	networking "github.com/quibbble/go-boardgame-networking"
	"github.com/quibbble/go-boardgame/pkg/bgn"
	"github.com/rs/zerolog"
	"github.com/unrolled/render"
	"net/http"
	"strings"
	"text/scanner"
	"time"
)

type Handler struct {
	log     zerolog.Logger
	render  *render.Render
	network *networking.GameNetwork
}

func NewHandler(log zerolog.Logger, render *render.Render, network *networking.GameNetwork) *Handler {
	return &Handler{
		log:     log,
		render:  render,
		network: network,
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
			Seed:        time.Now().UnixNano(),
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
	reader := strings.NewReader(load.BGN)
	sc := scanner.Scanner{}
	sc.Init(reader)
	game, err := bgn.Parse(&sc)
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
	if err := h.network.LoadGame(networking.LoadGameOptions{
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
		GameKey: gameKey,
		GameID:  gameID,
		Conn:    conn,
	}); err != nil {
		_ = conn.Close()
	}
}

func (h *Handler) GetBGN(w http.ResponseWriter, r *http.Request) {
	gameKey := r.URL.Query().Get("GameKey")
	gameID := r.URL.Query().Get("GameID")
	game, err := h.network.GetBGN(gameKey, gameID)
	if err != nil {
		writeJSONResponse(h.render, w, http.StatusInternalServerError, errorResponse{Message: err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(game.String()))
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
