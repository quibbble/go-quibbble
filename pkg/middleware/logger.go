package middleware

import (
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func RequestLogger(log zerolog.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				h.ServeHTTP(w, r)
				return
			}
			c := alice.New()
			c = c.Append(hlog.NewHandler(log))
			c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
				hlog.FromRequest(r).Info().
					Str("method", r.Method).
					Stringer("url", r.URL).
					Int("status", status).
					Msg("REQ")
			}))
			c = c.Append(hlog.RemoteAddrHandler("ip"))
			c = c.Append(hlog.RefererHandler("referer"))
			c.Then(h).ServeHTTP(w, r)
		})
	}
}
