package subscription_service

import (
	"context"
	"errors"
	"fmt"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"vk-subpub/internal/proto/proto"
	"vk-subpub/internal/subpub"
)

type server struct {
	proto.UnimplementedPubSubServer
	bus    *subpub.Bus
	logger *slog.Logger
}

func NewServer(b *subpub.Bus, l *slog.Logger) server {
	return server{bus: b, logger: l}
}

// Subscribe подписывает на событие
func (s server) Subscribe(req *proto.SubscribeRequest, stream proto.PubSub_SubscribeServer) error {
	s.logger.Info("Новая подписка: " + req.Key)

	// канал пришедших событий
	events := make(chan string, 24)
	sub, err := s.bus.Subscribe(req.Key, func(msg interface{}) {
		strMsg, ok := msg.(string)
		if !ok {
			s.logger.Warn("получено не строковое сообщение")
		} else {
			events <- strMsg
		}
	})
	if err != nil {
		s.logger.Warn(fmt.Sprintf("ошибка подписки: %v", err))
		return status.Errorf(codes.Internal, "ошибка подписки: %v", err)
	}
	defer sub.Unsubscribe()
	for {
		select {
		case data := <-events:
			if err := stream.Send(&proto.Event{Data: data}); err != nil {
				s.logger.Warn(fmt.Sprintf("ошибка отправки события: %v", err))
				return status.Errorf(codes.Unavailable, "не удалось отправить событие: %v", err)
			}
		case <-stream.Context().Done():
			s.logger.Warn(fmt.Sprintf("подписка отменена клиентом: %v", err))
			return status.Error(codes.Canceled, "подписка отменена клиентом")
		}
	}
}

// Publish принимает одно сообщение и раздаёт его всем подписчикам key
func (s server) Publish(ctx context.Context, req *proto.PublishRequest) (*emptypb.Empty, error) {
	if err := s.bus.Publish(req.Key, req.Data); err != nil {
		if errors.Is(err, subpub.ErrBusClosed) {
			s.logger.Warn(fmt.Sprintf("запись в закрытую шину: %v", err))
			return nil, status.Errorf(codes.Unavailable, "шина закрыта")
		}
		s.logger.Warn(fmt.Sprintf("ошибка публикации: %v", err))
		return nil, status.Errorf(codes.Internal, "ошибка публикации: %v", err)
	}
	return &emptypb.Empty{}, nil
}
