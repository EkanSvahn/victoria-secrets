package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	redisadapter "victora-secret-code/backend/internal/adapters/redis"
	applayer "victora-secret-code/backend/internal/app"
	httpadapter "victora-secret-code/backend/internal/adapters/http"
	"victora-secret-code/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err.Error())
		os.Exit(1)
	}

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err.Error())
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOptions)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping failed", "error", err.Error())
		os.Exit(1)
	}
	if cfg.StrictRedisEphemeral {
		if err := redisadapter.CheckEphemeralConfig(ctx, redisClient); err != nil {
			slog.Error("redis ephemeral self-check failed", "error", err.Error())
			os.Exit(1)
		}
	} else {
		if err := redisadapter.CheckEphemeralConfig(ctx, redisClient); err != nil {
			slog.Warn("redis ephemeral self-check warning", "error", err.Error())
		}
	}

	repo := redisadapter.New(redisClient)
	service := applayer.NewService(repo, cfg.MaxTTLSeconds, cfg.IDLengthBytes)
	server := httpadapter.NewServer(cfg, service)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "addr", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			slog.Error("server stopped", "error", err.Error())
			os.Exit(1)
		}
	}()

	<-shutdown
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = server.Shutdown(ctxShutdown)
	_ = redisClient.Close()
	slog.Info("server shutdown complete")
}
