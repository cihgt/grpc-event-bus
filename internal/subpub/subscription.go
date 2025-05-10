package subpub

import "sync"

type Subscription interface {
	Unsubscribe()
}

// subscription структура подписки, обрабатывающая события, пришедшие в локальный канал
type subscription struct {
	// поля нужные для unsubscribe
	bus     *Bus
	subject string

	msgHandler MessageHandler
	q          chan interface{}
	wg         *sync.WaitGroup
	once       *sync.Once
}

func NewSubscription(subject string, cb MessageHandler, bus *Bus) subscription {
	s := &subscription{
		subject:    subject,
		msgHandler: cb,
		bus:        bus,
		q:          make(chan interface{}, 42),
		wg:         &sync.WaitGroup{},
		once:       &sync.Once{},
	}
	s.wg.Add(1)
	go s.processMsg()
	return *s
}

func (s *subscription) processMsg() {
	defer s.wg.Done()
	for val := range s.q {
		s.msgHandler(val)
	}
}

func (s *subscription) Unsubscribe() {

	s.once.Do(func() {
		s.bus.mu.Lock()
		defer s.bus.mu.Unlock()
		if subs, ok := s.bus.mp[s.subject]; ok {
			for i, val := range subs {
				if val == s {
					s.bus.mp[s.subject] = append(s.bus.mp[s.subject][:i], s.bus.mp[s.subject][i+1:]...)
					break
				}

			}
		}
		close(s.q)
		s.wg.Wait()
	})
}

func (s *subscription) PushMsg(subject interface{}) {
	s.q <- subject
}
