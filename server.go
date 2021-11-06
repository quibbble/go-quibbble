package go_boardgame_networking

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer(port int, options GameNetworkOptions) *http.Server {
	network := NewGameNetwork(options)

	router := chi.NewRouter()
	router.Route("/game", func(router chi.Router) {
		router.Post("/create", func(w http.ResponseWriter, r *http.Request) {
			var options CreateGameOptions
			if err := unmarshalJSONRequestBody(r, &options); err != nil {
				http.Error(w, "failed to unmarshal create game options", http.StatusBadRequest)
				return
			}
			if err := network.CreateGame(options); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})

		router.Get("/join", func(w http.ResponseWriter, r *http.Request) {
			gameKey := r.URL.Query().Get("GameKey")
			gameID := r.URL.Query().Get("GameID")
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				http.Error(w, "failed to upgrade websocket connection", http.StatusInternalServerError)
				return
			}
			if err := network.JoinGame(JoinGameOptions{
				GameKey: gameKey,
				GameID:  gameID,
				Conn:    conn,
			}); err != nil {
				_ = conn.Close()
			}
		})
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
}
