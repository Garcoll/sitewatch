package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"example.com/sitewatch/internal/checker"
)

type Target struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type TargetStore struct {
	mu      sync.Mutex
	nextID  int
	targets []Target
}

type CheckResponse struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func main() {
	store := &TargetStore{
		nextID: 3,
		targets: []Target{
			{ID: 1, URL: "https://www.baidu.com", Name: "百度首页"},
			{ID: 2, URL: "https://github.com", Name: "GitHub 首页"},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", hello)
	mux.HandleFunc("GET /targets", store.getTargets)
	mux.HandleFunc("POST /targets", store.createTarget)
	mux.HandleFunc("POST /checks", store.checkTargets)

	fmt.Println("监听地址：127.0.0.1:8081")

	if err := http.ListenAndServe("127.0.0.1:8081", mux); err != nil {
		fmt.Println("服务停止：", err)
	}
}

func (s *TargetStore) snapshotTargets() []Target {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := make([]Target, len(s.targets))
	copy(snapshot, s.targets)
	return snapshot
}

func (s *TargetStore) getTargets(w http.ResponseWriter, r *http.Request) {
	targets := s.snapshotTargets()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

func (s *TargetStore) createTarget(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "请求正文必须是有效 JSON", http.StatusBadRequest)
		return
	}

	if input.URL == "" || input.Name == "" {
		http.Error(w, "url 和 name 都不能为空", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		http.Error(w, "url 格式不正确", http.StatusBadRequest)
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "url 必须使用 http 或 https", http.StatusBadRequest)
		return
	}

	if parsedURL.Hostname() == "" {
		http.Error(w, "url 必须包含主机名", http.StatusBadRequest)
		return
	}

	s.mu.Lock()

	target := Target{
		ID:   s.nextID,
		URL:  input.URL,
		Name: input.Name,
	}

	s.nextID++
	s.targets = append(s.targets, target)

	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(target); err != nil {
		fmt.Println("写入 JSON 响应失败：", err)
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Sitewatch")
}

func (s *TargetStore) checkTargets(w http.ResponseWriter, r *http.Request) {
	targets := s.snapshotTargets()

	urls := make([]string, len(targets))
	for i, target := range targets {
		urls[i] = target.URL
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	results := checker.CheckWebsites(client, urls, 3)

	responses := make([]CheckResponse, len(results))
	for i, result := range results {
		response := CheckResponse{
			ID:         targets[i].ID,
			Name:       targets[i].Name,
			URL:        result.URL,
			StatusCode: result.StatusCode,
			DurationMS: result.Duration.Milliseconds(),
		}
		if result.Err != nil {
			response.Error = result.Err.Error()
		}
		responses[i] = response
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}
