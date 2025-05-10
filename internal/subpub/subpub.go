package subpub

import (
	"context"
	"errors"
	"sync"
)

type MessageHandler func(msg interface{})

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

var ErrBusClosed = errors.New("event bus is closed")

type Bus struct {
	mu     sync.Mutex
	mp     map[string][]*subscription
	closed bool
}

func NewSubPub() *Bus {
	return &Bus{mp: make(map[string][]*subscription), closed: false}
}

func (b *Bus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if b.closed {
		return nil, ErrBusClosed
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	newSub := NewSubscription(subject, cb, b)
	b.mp[subject] = append(b.mp[subject], &newSub)
	return &newSub, nil
}

func (b *Bus) Publish(subject string, msg interface{}) error {
	if b.closed {
		return ErrBusClosed
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, sub := range b.mp[subject] {
		sub.PushMsg(msg)
	}
	return nil
}

func (b *Bus) Close(ctx context.Context) error {
	wg := sync.WaitGroup{}
	for _, subs := range b.mp {
		for i := range subs {
			wg.Add(1)
			go func() {
				subs[i].Unsubscribe()
				wg.Done()
			}()
		}
	}
	b.closed = true
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
