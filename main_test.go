package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

			result := checkWebsite(client, server.URL)

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

	result := checkWebsite(client, server.URL)

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
