package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	"github.com/okrammeitei/chatgo/internal/auth"
	"github.com/okrammeitei/chatgo/internal/config"
	"github.com/okrammeitei/chatgo/internal/conversation"
	"github.com/okrammeitei/chatgo/internal/file"
	"github.com/okrammeitei/chatgo/internal/message"
	"github.com/okrammeitei/chatgo/internal/notification"
	"github.com/okrammeitei/chatgo/internal/presence"
	"github.com/okrammeitei/chatgo/internal/search"
	"github.com/okrammeitei/chatgo/internal/user"
	"github.com/okrammeitei/chatgo/pkg/cache"
	"github.com/okrammeitei/chatgo/pkg/database"
	"github.com/okrammeitei/chatgo/pkg/logger"
	mw "github.com/okrammeitei/chatgo/pkg/middleware"
	ws "github.com/okrammeitei/chatgo/pkg/websocket"
)

// tokenValidatorAdapter bridges auth.Service to mw.TokenValidator
// without creating an import cycle.
type tokenValidatorAdapter struct {
	svc auth.Service
}

func (a *tokenValidatorAdapter) ValidateAccessToken(ctx context.Context, tokenStr string) (*mw.Claims, error) {
	c, err := a.svc.ValidateAccessToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}
	return &mw.Claims{
		UserID:    c.UserID,
		Username:  c.Username,
		RoleID:    c.RoleID,
		SessionID: c.SessionID,
	}, nil
}

func main() {
	configPath := os.Getenv("CHATGO_CONFIG")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.Production)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// ── Database ────────────────────────────────────────────────────────────
	db, err := database.NewPool(context.Background(), cfg.Database.DSN())
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// ── Redis ────────────────────────────────────────────────────────────────
	redisClient, err := cache.New(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer redisClient.Close()

	// ── WebSocket Hub ────────────────────────────────────────────────────────
	hub := ws.NewHub(log)
	go hub.Run()

	// ── Repositories ─────────────────────────────────────────────────────────
	authRepo := auth.NewPostgresRepository(db)
	userRepo := user.NewPostgresRepository(db)
	convRepo := conversation.NewPostgresRepository(db)
	msgRepo := message.NewPostgresRepository(db)
	notifRepo := notification.NewPostgresRepository(db)
	fileRepo := file.NewPostgresRepository(db)
	activityRepo := activitylog.NewPostgresRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────
	activitySvc := activitylog.NewService(activityRepo, log)
	notifSvc := notification.NewService(notifRepo, activitySvc, hub, log)
	authSvc := auth.NewService(authRepo, userRepo, activitySvc,
		cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL, log)

	defaultRoleID := user.DefaultRoleID()
	userSvc := user.NewService(userRepo, activitySvc, defaultRoleID, log)
	convSvc := conversation.NewService(convRepo, activitySvc, log)
	msgSvc := message.NewService(msgRepo, convRepo, notifSvc, activitySvc, hub, log)
	presenceSvc := presence.NewService(redisClient, hub, log)
	fileSvc := file.NewService(fileRepo, activitySvc, nil, cfg.File.StoragePath, cfg.File.MaxSizeBytes, log)
	searchSvc := search.NewService(db, log)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler := auth.NewHandler(authSvc, log)
	userHandler := user.NewHandler(userSvc, log)
	convHandler := conversation.NewHandler(convSvc, log)
	msgHandler := message.NewHandler(msgSvc, log)
	presenceHandler := presence.NewHandler(presenceSvc, log)
	notifHandler := notification.NewHandler(notifSvc, log)
	fileHandler := file.NewHandler(fileSvc, cfg.File.MaxSizeBytes, log)
	searchHandler := search.NewHandler(searchSvc, log)
	activityHandler := activitylog.NewHandler(activitySvc, log)

	// ── WebSocket upgrader ────────────────────────────────────────────────────
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(mw.Metrics)
	r.Use(mw.CORS(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, cfg.CORS.AllowedHeaders))
	r.Use(mw.RateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst, log))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		mw.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", promhttp.Handler())

	tokenValidator := &tokenValidatorAdapter{svc: authSvc}

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", authHandler.Routes())
		r.Post("/users", userHandler.Create)

		r.Group(func(r chi.Router) {
			r.Use(mw.Authenticator(tokenValidator, log))

			r.Mount("/users", userHandler.Routes())
			r.Mount("/conversations", convHandler.Routes())
			r.Mount("/notifications", notifHandler.Routes())
			r.Mount("/files", fileHandler.Routes())
			r.Mount("/search", searchHandler.Routes())
			r.Mount("/presence", presenceHandler.Routes())
			r.Mount("/activity-logs", activityHandler.Routes())
			r.Mount("/conversations/{convID}/messages", msgHandler.Routes())

			r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
				userID, ok := mw.UserIDFromCtx(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					log.Warn("ws upgrade failed", zap.Error(err))
					return
				}
				client := ws.NewClient(hub, conn, userID, log)
				hub.Register() <- client
				_ = presenceSvc.SetPresence(r.Context(), userID, &presence.UpdateRequest{Status: presence.StatusOnline})
				go client.WritePump()
				client.ReadPump(nil)
				_ = presenceSvc.SetOffline(r.Context(), userID)
			})
		})
	})

	// ── HTTP Server ───────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced shutdown", zap.Error(err))
	}
	log.Info("server stopped")
}
