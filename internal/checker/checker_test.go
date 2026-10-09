package checker

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCheckWebsiteStatusCode(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}),
			)
			defer server.Close()

			client := &http.Client{
				Timeout: time.Second,
			}

			result := CheckWebsite(client, server.URL)

			if result.Err != nil {
				t.Fatalf("预期收到 HTTP 响应，实际请求失败：%v", result.Err)
			}

			if result.StatusCode != statusCode {
				t.Errorf(
					"预期状态码 %d，实际为 %d",
					statusCode,
					result.StatusCode,
				)
			}

			if result.URL != server.URL {
				t.Errorf(
					"预期网址 %q，实际为 %q",
					server.URL,
					result.URL,
				)
			}
		})
	}
}

func TestCheckWebsiteTimeout(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 不发送响应，等待客户端因超时取消请求。
			<-r.Context().Done()
		}),
	)
	defer server.Close()

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	result := CheckWebsite(client, server.URL)

	if result.Err == nil {
		t.Fatal("预期请求超时，实际没有返回错误")
	}

	var timeoutErr interface {
		Timeout() bool
	}

	if !errors.As(result.Err, &timeoutErr) || !timeoutErr.Timeout() {
		t.Fatalf("预期超时错误，实际为：%v", result.Err)
	}
}

func TestCheckWebsitesPreservesInputOrder(t *testing.T) {
	releaseSlow := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow":
			select {
			case <-releaseSlow:
			case <-r.Context().Done():
				return
			}
		case "/release":
			close(releaseSlow)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// With two workers, /release starts only after /fast has sent its result.
	// This guarantees that results arrive out of input order without sleeps.
	targets := []string{server.URL + "/slow", server.URL + "/fast", server.URL + "/release"}
	client := &http.Client{Timeout: 5 * time.Second}
	results := CheckWebsites(client, targets, 2)

	if len(results) != len(targets) {
		t.Fatalf("expected %d results, got %d", len(targets), len(results))
	}
	for i, result := range results {
		if result.Err != nil {
			t.Fatalf("request %d failed: %v", i, result.Err)
		}
		if result.URL != targets[i] || result.Index != i {
			t.Errorf("result %d: got URL %q and index %d; want URL %q and index %d", i, result.URL, result.Index, targets[i], i)
		}
	}
}

func TestCheckWebsitesConcurrencyLimit(t *testing.T) {
	const concurrency = 2
	const total = 5

	var mu sync.Mutex
	active := 0
	peak := 0

	started := make(chan struct{}, total)
	release := make(chan struct{})

	// 1. 定义测试服务器收到请求后执行的逻辑。
	handler := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		if active > peak {
			peak = active
		}
		mu.Unlock()

		// 通知测试：这个请求已经进入服务端。
		started <- struct{}{}

		// 等待放行；如果客户端取消请求，也退出等待。
		select {
		case <-release:
		case <-r.Context().Done():
		}

		mu.Lock()
		active--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	var releaseOnce sync.Once
	unblock := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	defer unblock()

	// 3. 准备 5 个检查任务，都请求这个测试服务器。
	targets := make([]string, total)
	for i := range targets {
		targets[i] = server.URL
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 4. 后台运行检查，主流程继续负责等待和放行。
	done := make(chan []CheckResult, 1)
	go func() {
		done <- CheckWebsites(client, targets, concurrency)
	}()

	// 5. 等待两个请求进入服务端。
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	for i := 0; i < concurrency; i++ {
		select {
		case <-started:
			// 收到一次“请求已经进入”的通知。
		case <-timer.C:
			t.Fatal("等待两个请求同时进入服务端超时")
		}
	}

	// 两个请求已经同时停在服务端，现在放行。
	unblock()

	// 6. 等待全部结果。
	var results []CheckResult
	select {
	case results = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("等待批量检查完成超时")
	}

	// 7. 检查结果数量和每次请求是否成功。
	if len(results) != total {
		t.Fatalf("预期 %d 个结果，实际为 %d", total, len(results))
	}

	for i, result := range results {
		if result.Err != nil || result.StatusCode != http.StatusOK {
			t.Errorf(
				"请求 %d 未成功：状态码 %d，错误 %v",
				i, result.StatusCode, result.Err,
			)
		}
	}

	// 8. 检查本次运行中观察到的最大并发数。
	mu.Lock()
	gotPeak := peak
	mu.Unlock()

	if gotPeak != concurrency {
		t.Errorf(
			"预期最大并发数为 %d，实际为 %d",
			concurrency, gotPeak,
		)
	}
}
