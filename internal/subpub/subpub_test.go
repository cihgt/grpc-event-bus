package subpub

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := NewSubPub()
	defer bus.Close(context.Background())

	received := make(chan string, 3)
	// добавляем обработчик на топик
	sub, err := bus.Subscribe("topic", func(msg interface{}) {
		received <- msg.(string)
	})
	if err != nil {
		t.Fatalf("ошибка подписки: %v", err)
	}
	defer sub.Unsubscribe()

	// публикуем сообщения
	bus.Publish("topic", "a")
	bus.Publish("topic", "b")
	bus.Publish("topic", "c")

	// должны получить сообщения в том же порядке
	expected := []string{"a", "b", "c"}
	for _, exp := range expected {
		select {
		case got := <-received:
			if got != exp {
				t.Errorf("ожидалось %s, получили %s", exp, got)
			}
		case <-time.After(time.Second):
			t.Errorf("таймаут ожидания сообщения %s", exp)
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()
	defer bus.Close(context.Background())

	received := make(chan string, 2)
	// добавляем обработчик на топик
	sub, _ := bus.Subscribe("topic", func(msg interface{}) {
		received <- msg.(string)
	})

	bus.Publish("topic", "first")

	// отписываемся
	sub.Unsubscribe()

	// Publish more messages
	bus.Publish("topic", "second")
	bus.Publish("topic", "third")

	// должны получить только первое сообщение
	select {
	case got := <-received:
		if got != "first" {
			t.Errorf("ожидалось 'first', получили %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("таймаут ожидания первого сообщения")
	}

	select {
	case got := <-received:
		t.Errorf("неожиданное сообщение после отписки: %s", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestFIFOOrder(t *testing.T) {
	bus := NewSubPub()
	defer bus.Close(context.Background())

	received := make(chan int, 5)
	// добавляем обработчик на топик
	sub, _ := bus.Subscribe("numbers", func(msg interface{}) {
		received <- msg.(int)
	})
	defer sub.Unsubscribe()

	for i := 0; i < 5; i++ {
		bus.Publish("numbers", i)
	}

	// проверяем, что порядок сохранился
	for i := 0; i < 5; i++ {
		select {
		case got := <-received:
			if got != i {
				t.Errorf("ожидалось %d, получили %d", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("таймаут ожидания сообщения %d", i)
		}
	}
}

func TestSlowSubscriber(t *testing.T) {
	bus := NewSubPub()
	defer bus.Close(context.Background())

	fast := make(chan int, 3)
	slow := make(chan int, 3)

	// быстрый обработчик
	_, _ = bus.Subscribe("ping", func(msg interface{}) {
		fast <- msg.(int)
	})
	// медленный обработчик
	_, _ = bus.Subscribe("ping", func(msg interface{}) {
		time.Sleep(time.Second)
		slow <- msg.(int)
	})

	start := time.Now()
	for i := 1; i <= 3; i++ {
		bus.Publish("ping", i)
	}

	// должны получить сначала быстрых
	for i := 1; i <= 3; i++ {
		select {
		case got := <-fast:
			if got != i {
				t.Errorf("быстрый подписчик: ожидалось %d, получили %d", i, got)
			}
		case <-time.After(time.Second):
			t.Fatal("таймаут ожидания быстрого подписчика")
		}
	}
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		t.Errorf("быстрый подписчик заблокирован, заняло %v", elapsed)
	}

	// затем должны получить медленных
	for i := 1; i <= 3; i++ {
		select {
		case got := <-slow:
			if got != i {
				t.Errorf("медленный подписчик: ожидалось %d, получили %d", i, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("таймаут ожидания медленного подписчика")
		}
	}
}

func TestClose(t *testing.T) {
	bus := NewSubPub()
	received := make(chan string, 1)
	_, _ = bus.Subscribe("end", func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		received <- msg.(string)
	})

	bus.Publish("end", "bye")

	// закрытие с контекстом с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bus.Close(ctx); err != nil {
		t.Fatalf("ошибка закрытия: %v", err)
	}

	// Сообщение должно быть доставлено
	select {
	case got := <-received:
		if got != "bye" {
			t.Errorf("ожидалось 'bye', получили %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("таймаут ожидания сообщения после закрытия")
	}

	// должны получить ошибку после закрытия и истечения таймаута
	if err := bus.Publish("end", "later"); !errors.Is(err, ErrBusClosed) {
		t.Errorf("ожидалась ErrBusClosed, получили %v", err)
	}
}

func TestClose_ContextCanceledBeforeClose(t *testing.T) {
	bus := NewSubPub()

	// создаём контекст и сразу его отменяем
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// вызываем Close с уже отменённым контекстом
	err := bus.Close(ctx)
	if err == nil {
		t.Fatal("ожидалась ошибка при закрытии с отменённым контекстом, но получили nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидалась ошибка context.Canceled, но получили: %v", err)
	}
}

func TestPublishSubscribeToClosedBus(t *testing.T) {
	bus := NewSubPub()
	// закрываем шину сразу после обьявления
	bus.Close(context.Background())

	// подписываемся на топик в закрытой шине
	received := make(chan string, 3)
	_, err := bus.Subscribe("topic", func(msg interface{}) {
		received <- msg.(string)
	})
	// должны получить ошибку ErrBusClosed
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("ожидалась ErrBusClosed, получили %v", err)
	}
}
