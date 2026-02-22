package handlers

import (
	w "SelektorDisc/pkg/webrtc"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func (h *Handlers) Stream(c *fiber.Ctx) error {
	suuid := c.Params("suuid")
	if suuid == "" {
		c.Status(400)
		return nil
	}

	ws := "ws"

	if isProduction() {
		ws = "wss"
	}
	w.RoomsLock.RLock()
	if _, ok := w.Streams[suuid]; ok {
		w.RoomsLock.RUnlock()
		turnUsername, turnPassword := turnCredentials()
		return c.Render("stream", fiber.Map{
			"StreamWebsocketAddr": fmt.Sprintf("%s://%s/stream/%s/websocket", ws, c.Hostname(), suuid),
			"ChatWebsocketAddr":   fmt.Sprintf("%s://%s/stream/%s/chat/websocket", ws, c.Hostname(), suuid),
			"ViewerWebsocketAddr": fmt.Sprintf("%s://%s/stream/%s/viewer/websocket", ws, c.Hostname(), suuid),
			"TurnHost":            turnHost(),
			"TurnPort":            turnPort(),
			"TurnUsername":        turnUsername,
			"TurnPassword":        turnPassword,
			"Type":                "stream",
		}, "layouts/main")
	}
	w.RoomsLock.RUnlock()
	return c.Render("stream", fiber.Map{
		"NoStream": "true",
		"Leave":    "true",
	}, "layouts/main")
}

func (h *Handlers) StreamWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}
	w.RoomsLock.RLock()
	if stream, ok := w.Streams[suuid]; ok {
		w.RoomsLock.RUnlock()
		w.StreamConn(c, stream.Peers)
		return
	}
	w.RoomsLock.RUnlock()

}

func (h *Handlers) StreamViewerWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}
	w.RoomsLock.RLock()

	if stream, ok := w.Streams[suuid]; ok {
		w.RoomsLock.RUnlock()
		viewerConn(c, stream.Peers)
		return
	}
	w.RoomsLock.RUnlock()

}

func viewerConn(c *websocket.Conn, p *w.Peers) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case <-ticker.C:
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(fmt.Sprintf("%d", p.CountConnections())))
		}
	}
}
