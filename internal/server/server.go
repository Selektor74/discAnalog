package server

import (
	"flag"
	"os"
	"time"

	"SelektorDisc/internal/handlers"
	"SelektorDisc/internal/middleware"
	w "SelektorDisc/pkg/webrtc"

	"context"
	"log"

	"SelektorDisc/internal/repository"
	"SelektorDisc/internal/repository/chat"
	"SelektorDisc/internal/repository/rooms"
	"SelektorDisc/internal/repository/users"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	PostgresUser     = os.Getenv("POSTGRES_USER")
	PostgresPassword = os.Getenv("POSTGRES_PASSWORD")
	PostgresHost     = os.Getenv("POSTGRES_HOST")
	PostgresPort     = os.Getenv("POSTGRES_PORT")
	PostgresDB       = os.Getenv("POSTGRES_DB")
)

var (
	addr = flag.String("addr", ":"+os.Getenv("PORT"), "")
	cert = flag.String("cert", "", "")
	key  = flag.String("key", "", "")
)

func requireEnv(keys ...string) error {
	for _, key := range keys {
		if os.Getenv(key) == "" {
			return fmt.Errorf("required env var %s is not set", key)
		}
	}
	return nil
}

func Run() error {

	flag.Parse()
	if *addr == ":" {
		*addr = ":8080"
	}
	if err := requireEnv("AUTH_TOKEN_SECRET"); err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := newPostgresConnection(ctx)
	if err != nil {
		return fmt.Errorf("cannot connect to db:%w", err)
	}
	defer pool.Close()
	baseRepo := repository.NewBaseRepository(pool)
	usersRepo := users.NewUsersRepository(baseRepo)
	chatRepo := chat.NewChatRepository(baseRepo)
	roomsRepo := rooms.NewRoomsRepository(baseRepo)

	h := handlers.New(usersRepo, roomsRepo, chatRepo)

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{Views: engine})
	app.Use(logger.New())
	app.Use(cors.New())
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	authMiddleware := middleware.NewAuthMiddleware(os.Getenv("AUTH_TOKEN_SECRET"))
	defer func() {
		if err := authMiddleware.Close(); err != nil {
			log.Printf("auth middleware close error: %v", err)
		}
	}()

	app.Get("/", h.Welcome)
	app.Get("/register", h.Register)
	app.Post("/auth/login", h.Login)
	app.Post("/auth/register", h.Signup)
	app.Post("/auth/logout", h.Logout)
	app.Get("/rooms", authMiddleware.RequireAuth, h.Rooms)
	app.Get("/room/create", authMiddleware.RequireAuth, h.RoomCreate)
	app.Post("/room/create", authMiddleware.RequireAuth, h.RoomCreateWithName)
	app.Get("/room/:uuid", authMiddleware.RequireAuth, h.Room)
	app.Get("/room/:uuid/websocket", authMiddleware.RequireAuth, websocket.New(h.RoomWebsocket, websocket.Config{
		HandshakeTimeout: 20 * time.Second,
	}))
	app.Get("/room/:uuid/chat/websocket", authMiddleware.RequireAuth, websocket.New(h.RoomChatWebsocket))
	app.Get("/room/:uuid/viewer/websocket", authMiddleware.RequireAuth, websocket.New(h.RoomViewerWebsocket))
	app.Get("/stream/:ssuid", authMiddleware.RequireAuth, h.Stream)
	app.Get("/stream/:ssuid/websocket", authMiddleware.RequireAuth, websocket.New(h.StreamWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second}))
	app.Get("/stream/:ssuid/chat/websocket", authMiddleware.RequireAuth, websocket.New(h.StreamChatWebWebsocket))
	app.Get("/stream/:ssuid/viewer/websocket", authMiddleware.RequireAuth, websocket.New(h.StreamViewerWebsocket))
	app.Static("/", "./assets")

	w.RoomsLock.Lock()
	w.Rooms = make(map[string]*w.RoomRTC)
	w.Streams = make(map[string]*w.RoomRTC)
	w.RoomsLock.Unlock()

	go dispatchKeyFrames()

	if *cert != "" {
		return app.ListenTLS(*addr, *cert, *key)
	}
	return app.Listen(*addr)
}

func dispatchKeyFrames() {
	for range time.NewTicker(time.Second * 3).C {
		w.RoomsLock.RLock()
		rooms := make([]*w.RoomRTC, 0, len(w.Rooms))
		for _, room := range w.Rooms {
			rooms = append(rooms, room)
		}
		w.RoomsLock.RUnlock()

		for _, room := range rooms {
			room.Peers.DispatchKeyFrame()
		}
	}

}
func newPostgresConnection(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=voicechat",
		PostgresUser, PostgresPassword, PostgresHost, PostgresPort, PostgresDB,
	)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config error:%w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect:%w", err)
	}
	return pool, nil

}
