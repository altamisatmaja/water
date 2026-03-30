package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws upgrade", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch := make(chan *capture.Event, 100)
	go func() {
		_ = capture.Tail(ctx, s.eventsPath, ch)
		close(ch)
	}()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
