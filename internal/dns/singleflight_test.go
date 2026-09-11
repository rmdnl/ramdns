package dns

import (
	"sync"
	"testing"
	"time"
)

func TestSingleFlight(t *testing.T) {
	sf := NewSingleFlight()

	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)

	const requests = 20

	results := make([]interface{}, requests)

	wg.Add(requests)

	for i := 0; i < requests; i++ {
		go func(index int) {
			defer wg.Done()

			results[index] = sf.Do("google.com", func() interface{} {
				mu.Lock()
				counter++
				mu.Unlock()

				time.Sleep(50 * time.Millisecond)

				return "success"
			})
		}(i)
	}

	wg.Wait()

	if counter != 1 {
		t.Fatalf(
			"expected function to run once, ran %d times",
			counter,
		)
	}

	for i, result := range results {
		if result != "success" {
			t.Fatalf(
				"request %d returned %v",
				i,
				result,
			)
		}
	}
}
