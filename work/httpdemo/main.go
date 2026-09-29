package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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

func (s *TargetStore) getTargets(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.targets)
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

	fmt.Println("监听地址：127.0.0.1:8081")

	if err := http.ListenAndServe("127.0.0.1:8081", mux); err != nil {
		fmt.Println("服务停止：", err)
	}
}
