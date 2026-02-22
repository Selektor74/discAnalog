package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"SelektorDisc/internal/domain/models"
	"SelektorDisc/pkg/chat"
	w "SelektorDisc/pkg/webrtc"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

const (
	chatHistoryLimit = 50
	stickerPrefix    = "[[sticker:"
	stickerSuffix    = "]]"
)

type chatWireMessage struct {
	Author string `json:"author"`
	Time   string `json:"time"`
	Text   string `json:"text"`
}

func (h *Handlers) RoomChatWebsocket(c *websocket.Conn) {
	roomUUID := c.Params("uuid")
	if roomUUID == "" {
		return
	}

	w.RoomsLock.RLock()
	room := w.Rooms[roomUUID]
	w.RoomsLock.RUnlock()
	if room == nil || room.Hub == nil {
		return
	}

	username := h.resolveChatUsername(c)
	initialMessages := h.loadInitialMessages(roomUUID)
	onMessage := h.persistMessageHandler(roomUUID, username)

	chat.PeerChatConn(c.Conn, room.Hub, initialMessages, onMessage)
}

func (h *Handlers) StreamChatWebWebsocket(c *websocket.Conn) {
	suuid := c.Params("suuid")
	if suuid == "" {
		return
	}

	w.RoomsLock.RLock()
	stream, ok := w.Streams[suuid]
	w.RoomsLock.RUnlock()
	if !ok {
		return
	}

	w.RoomsLock.Lock()
	if stream.Hub == nil {
		hub := chat.NewHub()
		stream.Hub = hub
		go hub.Run()
	}
	hub := stream.Hub
	roomID := stream.RoomID
	w.RoomsLock.Unlock()

	if roomID == "" {
		return
	}

	username := h.resolveChatUsername(c)
	initialMessages := h.loadInitialMessages(roomID)
	onMessage := h.persistMessageHandler(roomID, username)

	chat.PeerChatConn(c.Conn, hub, initialMessages, onMessage)
}

func (h *Handlers) loadInitialMessages(roomID string) [][]byte {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messages, err := h.chatRepo.GetChatMessages(ctx, roomID, chatHistoryLimit, 0)
	if err != nil {
		return nil
	}

	out := make([][]byte, 0, len(messages))
	for _, message := range messages {
		if message == nil || strings.TrimSpace(message.Content) == "" {
			continue
		}
		out = append(out, buildChatWirePayload(message.Username, message.Date, message.Content))
	}
	return out
}

func (h *Handlers) persistMessageHandler(roomID, username string) func([]byte) []byte {
	displayName := strings.TrimSpace(username)
	if displayName == "" {
		displayName = "anonymous"
	}

	roomUUID, err := uuid.Parse(roomID)
	if err != nil {
		return func(raw []byte) []byte {
			content := strings.TrimSpace(string(raw))
			if content == "" {
				return nil
			}
			return buildChatWirePayload(displayName, time.Now().UTC(), content)
		}
	}

	return func(raw []byte) []byte {
		content := strings.TrimSpace(string(raw))
		if content == "" {
			return nil
		}
		if isStickerPayload(content) {
			// Sticker events are ephemeral: play sound for online peers, but don't store in chat history.
			return buildChatWirePayload(displayName, time.Now().UTC(), content)
		}

		now := time.Now().UTC()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, _ = h.chatRepo.CreateMessage(ctx, &models.Chat{
			Id:       uuid.New(),
			RoomId:   roomUUID,
			Content:  content,
			Date:     now,
			Username: displayName,
		})

		return buildChatWirePayload(displayName, now, content)
	}
}

func isStickerPayload(text string) bool {
	return strings.HasPrefix(text, stickerPrefix) && strings.HasSuffix(text, stickerSuffix)
}

func (h *Handlers) resolveChatUsername(c *websocket.Conn) string {
	rawUserID := c.Locals("user_uuid")
	userID, ok := rawUserID.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "anonymous"
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return "anonymous"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	user, err := h.usersRepo.GetUser(ctx, parsedUserID)
	if err != nil || user == nil || strings.TrimSpace(user.Username) == "" {
		return "anonymous"
	}
	return user.Username
}

func buildChatWirePayload(author string, at time.Time, text string) []byte {
	author = strings.TrimSpace(author)
	if author == "" {
		author = "anonymous"
	}

	payload := chatWireMessage{
		Author: author,
		Time:   at.Local().Format("15:04"),
		Text:   text,
	}

	out, err := json.Marshal(payload)
	if err != nil {
		return []byte(text)
	}
	return out
}
