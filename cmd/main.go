package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"vk-subpub/internal/config"
	"vk-subpub/internal/proto/proto"
	"vk-subpub/internal/subpub"
	"vk-subpub/internal/subscription-service"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(handler)

	configPath := "../config.json"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("error loading config: %v", err)
		os.Exit(1)
	}

	addr := ":" + strconv.Itoa(cfg.Port)

	// создаем шину событий
	bus := subpub.NewSubPub()

	// регистрируем сервис
	srv := grpc.NewServer()
	proto.RegisterPubSubServer(srv, subscription_service.NewServer(bus, logger))
	reflection.Register(srv)
	
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("не удалось слушать порт")
		os.Exit(1)
	}

	// graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		logger.Info("Получен сигнал завершения, останавливаем сервер")
		srv.GracefulStop()

		// ждём максимум 5 секунд, пока все подписчики отпишутся
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bus.Close(ctx); err != nil {
			logger.Error("ошибка закрытия шины")
		}
	}()

	logger.Info("gRPC сервер запущен на порту " + addr)
	if err := srv.Serve(lis); err != nil {
		logger.Error("ошибка при Serve()")
		os.Exit(1)
	}
}
