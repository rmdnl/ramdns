package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicHTTPHandler(t *testing.T) {
	doh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("doh"))
	})

	dashboard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dashboard"))
	})

	handler := publicHTTPHandler(doh, dashboard)

	tests := []struct {
		name       string
		host       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "DoH hostname",
			host:       "doh.ramdns.my.id",
			wantStatus: http.StatusOK,
			wantBody:   "doh",
		},
		{
			name:       "DoH hostname with port",
			host:       "doh.ramdns.my.id:443",
			wantStatus: http.StatusOK,
			wantBody:   "doh",
		},
		{
			name:       "DoH hostname trailing dot",
			host:       "doh.ramdns.my.id.",
			wantStatus: http.StatusOK,
			wantBody:   "doh",
		},
		{
			name:       "Dashboard hostname",
			host:       "dashboard.ramdns.my.id",
			wantStatus: http.StatusOK,
			wantBody:   "dashboard",
		},
		{
			name:       "Dashboard hostname with port",
			host:       "dashboard.ramdns.my.id:443",
			wantStatus: http.StatusOK,
			wantBody:   "dashboard",
		},
		{
			name:       "Unknown hostname",
			host:       "example.com",
			wantStatus: http.StatusMisdirectedRequest,
			wantBody:   "misdirected request\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://"+tt.host+"/", nil)
			req.Host = tt.host

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if rec.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
