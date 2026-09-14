package dashboard

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	apiClient *http.Client
	apiURL    string
	token     string
	staticDir string
}

func NewHandler(apiURL, token, staticDir string) *Handler {
	return &Handler{
		apiClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:    strings.TrimRight(apiURL, "/"),
		token:     token,
		staticDir: staticDir,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		h.proxyAPI(w, r)
		return
	}

	if r.URL.Path == "/" {
		http.ServeFile(w, r, filepath.Join(h.staticDir, "index.html"))
		return
	}

	http.FileServer(http.Dir(h.staticDir)).ServeHTTP(w, r)
}

func (h *Handler) proxyAPI(w http.ResponseWriter, r *http.Request) {
	target, err := url.JoinPath(h.apiURL, r.URL.Path)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Accept", "application/json")

	resp, err := h.apiClient.Do(req)
	if err != nil {
		http.Error(w, "management API unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func StaticDir() string {
	if value := strings.TrimSpace(os.Getenv("RAMDNS_DASHBOARD_DIR")); value != "" {
		return value
	}
	return filepath.Join("web", "dashboard")
}
