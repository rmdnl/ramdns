package upstream

import "testing"

func TestServerHealthTransitions(t *testing.T) {
	s := &Server{}
	s.healthy.Store(true)

	s.markFailure()
	s.markFailure()

	if !s.isHealthy() {
		t.Fatal("server became unhealthy before failure threshold")
	}

	s.markFailure()

	if s.isHealthy() {
		t.Fatal("server did not become unhealthy after failure threshold")
	}

	s.markSuccess()

	if s.isHealthy() {
		t.Fatal("server recovered before success threshold")
	}

	s.markSuccess()

	if !s.isHealthy() {
		t.Fatal("server did not recover after success threshold")
	}
}

func TestHealthyServersFallback(t *testing.T) {
	healthy := &Server{Addr: "healthy"}
	healthy.healthy.Store(true)

	unhealthy := &Server{Addr: "unhealthy"}
	unhealthy.healthy.Store(false)

	r := &Resolver{
		Servers: []*Server{healthy, unhealthy},
	}

	got := r.healthyServers()

	if len(got) != 1 || got[0] != healthy {
		t.Fatalf("expected only healthy server, got %#v", got)
	}

	healthy.healthy.Store(false)

	got = r.healthyServers()

	if len(got) != 2 {
		t.Fatalf("expected fallback to all servers, got %d", len(got))
	}
}
