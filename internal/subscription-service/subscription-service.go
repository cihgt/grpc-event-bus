package subscription_service

import (
	"context"
	"errors"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"vk-subpub/internal/proto/proto"
	"vk-subpub/internal/subpub"
)

type service struct {
	proto.UnimplementedPubSubServer
	bus    *subpub.Bus
	logger *slog.Logger
}

func NewService(b *subpub.Bus, l *slog.Logger) service {
	return service{bus: b, logger: l}
}

// Subscribe подписывает на событие
func (s service) Subscribe(req *proto.SubscribeRequest, stream proto.PubSub_SubscribeServer) error {
	s.logger.Info("Получена новая подписка", slog.String("key", req.Key))

	// канал пришедших событий
	events := make(chan string, 24)
	sub, err := s.bus.Subscribe(req.Key, func(msg interface{}) {
		strMsg, ok := msg.(string)
		if !ok {
			s.logger.Warn("Получено не строковое сообщение",
				slog.String("key", req.Key),
				slog.Any("message", msg),
			)
		} else {
			events <- strMsg
		}
	})
	if err != nil {
		s.logger.Error("Ошибка при подписке",
			slog.String("key", req.Key),
			slog.Any("error", err),
		)
		return status.Errorf(codes.Internal, "ошибка подписки: %v", err)
	}
	defer sub.Unsubscribe()
	for {
		select {
		case data := <-events:
			if err := stream.Send(&proto.Event{Data: data}); err != nil {
				s.logger.Error("Ошибка отправки события",
					slog.String("key", req.Key),
					slog.Any("error", err),
				)
				return status.Errorf(codes.Unavailable, "не удалось отправить событие: %v", err)
			}
		case <-stream.Context().Done():
			s.logger.Debug("Подписка отменена клиентом", slog.String("key", req.Key))
			return status.Error(codes.Canceled, "подписка отменена клиентом")
		}
	}
}

// Publish принимает одно сообщение и раздаёт его всем подписчикам key
func (s service) Publish(ctx context.Context, req *proto.PublishRequest) (*emptypb.Empty, error) {
	if err := s.bus.Publish(req.Key, req.Data); err != nil {
		if errors.Is(err, subpub.ErrBusClosed) {
			s.logger.Error("Попытка записи в закрытую шину",
				slog.String("key", req.Key),
				slog.Any("error", err),
			)
			return nil, status.Errorf(codes.Unavailable, "шина закрыта")
		}
		s.logger.Error("Ошибка публикации сообщения",
			slog.String("key", req.Key),
			slog.String("data", req.Data),
			slog.Any("error", err),
		)
		return nil, status.Errorf(codes.Internal, "ошибка публикации: %v", err)
	}
	s.logger.Info("Сообщение успешно опубликовано",
		slog.String("key", req.Key),
		slog.String("data", req.Data),
	)
	return &emptypb.Empty{}, nil
}
