package dns

import "sync"

type Flight struct {
	wg  sync.WaitGroup
	msg interface{}
}

type SingleFlight struct {
	mu      sync.Mutex
	flights map[string]*Flight
}

func NewSingleFlight() *SingleFlight {
	return &SingleFlight{
		flights: make(map[string]*Flight),
	}
}

func (s *SingleFlight) Do(
	key string,
	fn func() interface{},
) interface{} {
	s.mu.Lock()

	if flight, ok := s.flights[key]; ok {
		s.mu.Unlock()

		flight.wg.Wait()

		return flight.msg
	}

	flight := &Flight{}
	flight.wg.Add(1)

	s.flights[key] = flight

	s.mu.Unlock()

	flight.msg = fn()
	flight.wg.Done()

	s.mu.Lock()
	delete(s.flights, key)
	s.mu.Unlock()

	return flight.msg
}
