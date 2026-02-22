package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	htemplate "html/template"
	"log"
	"strings"
	"time"

	"SelektorDisc/internal/domain/models"
	"SelektorDisc/pkg/chat"
	w "SelektorDisc/pkg/webrtc"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	guuid "github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

type roomCreateRequest struct {
	Name string `json:"name" form:"name"`
}

func (h *Handlers) RoomCreate(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, roomUUID, _, err := h.createRoom(ctx, "Новая комната")
	if err != nil {
		return err
	}

	return c.Redirect(fmt.Sprintf("/room/%s", roomUUID))
}

func (h *Handlers) RoomCreateWithName(c *fiber.Ctx) error {
	var req roomCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Новая комната"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, roomUUID, _, err := h.createRoom(ctx, name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot create room"})
	}

	return c.JSON(fiber.Map{
		"room_uuid": roomUUID,
		"room_name": name,
	})
}

func (h *Handlers) Room(c *fiber.Ctx) error {
	ctx := c.Context()
	roomUUID := c.Params("uuid")
	if roomUUID == "" {
		c.Status(400)
		return nil
	}

	ws := "ws"
	if isProduction() {
		ws = "wss"
	}

	uuid, suuid, roomRTC, err := h.getRoom(roomUUID, ctx)
	if err != nil {
		return err
	}

	log.Printf("room is created or received -%s", uuid)

	roomName := "Комната"
	if roomRTC != nil && roomRTC.Name != "" {
		roomName = roomRTC.Name
	}

	localUsername := h.resolveCurrentUsername(c)
	turnUsername, turnPassword := turnCredentials()

	return c.Render("peer", fiber.Map{
		"RoomWebsocketAddr":   fmt.Sprintf("%s://%s/room/%s/websocket", ws, c.Hostname(), uuid),
		"RoomLink":            fmt.Sprintf("%s://%s/room/%s", c.Protocol(), c.Hostname(), uuid),
		"ChatWebsocketAddr":   fmt.Sprintf("%s://%s/room/%s/chat/websocket", ws, c.Hostname(), uuid),
		"ViewerWebsocketAddr": fmt.Sprintf("%s://%s/room/%s/viewer/websocket", ws, c.Hostname(), uuid),
		"StreamLink":          fmt.Sprintf("%s://%s/stream/%s", c.Protocol(), c.Hostname(), suuid),
		"RoomName":            roomName,
		"LocalUsername":       localUsername,
		"TurnHost":            turnHost(),
		"TurnPort":            turnPort(),
		"TurnUsername":        turnUsername,
		"TurnPassword":        turnPassword,
		"Type":                "room",
	}, "layouts/main")
}

func (h *Handlers) resolveCurrentUsername(c *fiber.Ctx) string {
	const fallbackUsername = "Вы"

	rawUserID := c.Locals("user_uuid")
	userID, ok := rawUserID.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return fallbackUsername
	}

	parsedUserID, err := guuid.Parse(userID)
	if err != nil {
		return fallbackUsername
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	user, err := h.usersRepo.GetUser(ctx, parsedUserID)
	if err != nil || user == nil || strings.TrimSpace(user.Username) == "" {
		return fallbackUsername
	}

	return user.Username
}

func (h *Handlers) Rooms(c *fiber.Ctx) error {
	roomsList, err := h.roomsRepo.GetAllRooms(c.Context())
	if err != nil {
		return err
	}

	roomsJSONBytes, err := json.Marshal(roomsList)
	if err != nil {
		return err
	}

	return c.Render("rooms", fiber.Map{
		"RoomsJSON": htemplate.JS(roomsJSONBytes),
	})
}

func (h *Handlers) RoomWebsocket(c *websocket.Conn) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _, room, err := h.getRoom(uuid, ctx)
	if err != nil {
		return
	}

	w.RoomConn(c, room.Peers)
}

func (handlers *Handlers) createRoom(ctx context.Context, nameOfRoom string) (string, string, *w.RoomRTC, error) {
	roomUUID := guuid.New()

	h := sha256.New()
	h.Write([]byte(roomUUID.String()))
	suuid := fmt.Sprintf("%x", h.Sum(nil))

	chatHub := chat.NewHub()
	p := &w.Peers{
		TrackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
		TrackOwners: make(map[string]*webrtc.PeerConnection),
	}
	room := &w.RoomRTC{
		RoomID: roomUUID.String(),
		Name:   nameOfRoom,
		Peers:  p,
		Hub:    chatHub,
	}

	newRoom, err := handlers.roomsRepo.CreateRoom(ctx, &models.Room{UUID: roomUUID, Name: nameOfRoom})
	if err != nil {
		return "", "", nil, fmt.Errorf("Failed to create new room:%w", err)
	}

	w.RoomsLock.Lock()
	w.Rooms[newRoom.UUID.String()] = room
	w.Streams[suuid] = room
	w.RoomsLock.Unlock()

	go chatHub.Run()
	return suuid, newRoom.UUID.String(), room, nil
}

func (handlers *Handlers) getRoom(roomUUID string, ctx context.Context) (string, string, *w.RoomRTC, error) {
	h := sha256.New()
	h.Write([]byte(roomUUID))
	suuid := fmt.Sprintf("%x", h.Sum(nil))

	w.RoomsLock.RLock()
	if room := w.Rooms[roomUUID]; room != nil {
		w.RoomsLock.RUnlock()
		w.RoomsLock.Lock()
		if _, ok := w.Streams[suuid]; !ok {
			w.Streams[suuid] = room
		}
		w.RoomsLock.Unlock()
		return roomUUID, suuid, room, nil
	}
	w.RoomsLock.RUnlock()

	existsRoom, err := handlers.roomsRepo.GetRoom(ctx, roomUUID)
	if err != nil {
		return "", "", nil, errors.New("Failed to get exist room")
	}
	if existsRoom == nil {
		return "", "", nil, errors.New("Room not found")
	}

	hub := chat.NewHub()
	p := &w.Peers{
		TrackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
		TrackOwners: make(map[string]*webrtc.PeerConnection),
	}
	room := &w.RoomRTC{
		RoomID: existsRoom.UUID.String(),
		Name:   existsRoom.Name,
		Peers:  p,
		Hub:    hub,
	}

	w.RoomsLock.Lock()
	if existing := w.Rooms[existsRoom.UUID.String()]; existing != nil {
		w.RoomsLock.Unlock()
		return existsRoom.UUID.String(), suuid, existing, nil
	}
	w.Rooms[existsRoom.UUID.String()] = room
	w.Streams[suuid] = room
	w.RoomsLock.Unlock()

	go hub.Run()
	return existsRoom.UUID.String(), suuid, room, nil
}

func (h *Handlers) RoomViewerWebsocket(c *websocket.Conn) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}

	w.RoomsLock.RLock()
	if peer, ok := w.Rooms[uuid]; ok {
		w.RoomsLock.RUnlock()
		roomViewerConn(c, peer.Peers)
		return
	}
	w.RoomsLock.RUnlock()
}

func roomViewerConn(c *websocket.Conn, p *w.Peers) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case <-ticker.C:
			writer, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = writer.Write([]byte(fmt.Sprintf("%d", p.CountConnections())))
		}
	}
}

