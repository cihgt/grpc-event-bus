package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
	"vk-subpub/internal/config"
	"vk-subpub/internal/proto/proto"

	"google.golang.org/grpc"
)

const (
	topicCount       = 100
	messagesPerTopic = 1000
	goPerTopic       = 50 // количество горутин на топик
)

func publishMessages(client proto.PubSubClient, topic string, msgCount int, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < msgCount; i++ {

		message := fmt.Sprintf("Сообщение %d из топика %s", i, topic)
		_, err := client.Publish(ctx, &proto.PublishRequest{
			Key:  topic,
			Data: message,
		})
		if err != nil {
			slog.Warn("Ошибка публикации", slog.String("topic", topic), slog.Any("error", err))
		} else {
			slog.Info("Сообщение опубликовано", slog.String("topic", topic), slog.String("message", message))
		}

		// случайная задержка для симуляции нагрузки
		time.Sleep(time.Millisecond * 100 * time.Duration(rand.Intn(10)))
	}
}

func main() {

	configPath := "../../config.json"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		slog.Error("Ошибка загрузки конфига", slog.Any("error", err))
		os.Exit(1)
	}

	addr := ":" + strconv.Itoa(cfg.Port)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		slog.Error("Ошибка подключения к gRPC серверу", slog.Any("error", err))
		os.Exit(1)
	}
	defer conn.Close()

	client := proto.NewPubSubClient(conn)

	var wg sync.WaitGroup

	start := time.Now()
	for t := 0; t < topicCount; t++ {
		topicName := fmt.Sprintf("topic-%d", t)
		for c := 0; c < goPerTopic; c++ {
			wg.Add(1)
			go publishMessages(client, topicName, messagesPerTopic/goPerTopic, &wg)
		}
	}

	// ожидание завершения всех публикаций
	wg.Wait()
	elapsed := time.Since(start)

	slog.Info(fmt.Sprintf("Завершено создание %d топиков с %d сообщениями в каждом за %s", topicCount, messagesPerTopic, elapsed))
}
