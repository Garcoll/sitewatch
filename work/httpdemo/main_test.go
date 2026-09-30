package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTargetValidation(t *testing.T) {
	store := &TargetStore{
		nextID: 3,
		targets: []Target{
			{ID: 1, URL: "https://www.baidu.com", Name: "百度首页"},
			{ID: 2, URL: "https://github.com", Name: "GitHub 首页"},
		},
	}

	// 第一次请求：非法网址应该被拒绝。
	badRequest := httptest.NewRequest(
		http.MethodPost,
		"/targets",
		strings.NewReader(`{"url":"banana","name":"错误网址测试"}`),
	)
	badRequest.Header.Set("Content-Type", "application/json")

	badResponse := httptest.NewRecorder()

	store.createTarget(badResponse, badRequest)

	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf(
			"非法网址：预期状态码 400，实际为 %d，正文：%s",
			badResponse.Code,
			badResponse.Body.String(),
		)
	}

	if len(store.targets) != 2 {
		t.Fatalf("非法请求不应增加目标，实际数量为 %d", len(store.targets))
	}

	if store.nextID != 3 {
		t.Fatalf("非法请求不应消耗 ID，实际 nextID 为 %d", store.nextID)
	}

	// 第二次请求：合法网址应该成功创建，仍然获得 ID 3。
	goodRequest := httptest.NewRequest(
		http.MethodPost,
		"/targets",
		strings.NewReader(`{"url":"https://go.dev","name":"Go 官网"}`),
	)
	goodRequest.Header.Set("Content-Type", "application/json")

	goodResponse := httptest.NewRecorder()

	store.createTarget(goodResponse, goodRequest)

	if goodResponse.Code != http.StatusCreated {
		t.Fatalf(
			"合法网址：预期状态码 201，实际为 %d，正文：%s",
			goodResponse.Code,
			goodResponse.Body.String(),
		)
	}

	var created Target

	if err := json.NewDecoder(goodResponse.Body).Decode(&created); err != nil {
		t.Fatalf("解析响应 JSON 失败：%v", err)
	}

	if created.ID != 3 {
		t.Errorf("预期新目标 ID 为 3，实际为 %d", created.ID)
	}

	if created.URL != "https://go.dev" || created.Name != "Go 官网" {
		t.Errorf("新目标内容不符合预期：%+v", created)
	}

	if len(store.targets) != 3 {
		t.Fatalf("成功添加后应有 3 个目标，实际为 %d", len(store.targets))
	}

	if store.targets[2] != created {
		t.Errorf("保存的目标与响应不一致：保存=%+v，响应=%+v",
			store.targets[2], created)
	}

	if store.nextID != 4 {
		t.Errorf("成功添加后 nextID 应为 4，实际为 %d", store.nextID)
	}
}

func TestCreateTargetRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "不支持的协议",
			body: `{"url":"banana","name":"测试目标"}`,
		},
		{
			name: "空名称",
			body: `{"url":"https://go.dev","name":""}`,
		},
		{
			name: "缺少主机名",
			body: `{"url":"https:///docs","name":"测试目标"}`,
		},
		{
			name: "错误JSON",
			body: `{"url":}`,
		},
		{
			name: "网址为空，但名称不为空",
			body: `{"url":"", "name":"测试目标"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := Target{
				ID:   1,
				URL:  "https://www.baidu.com",
				Name: "百度首页",
			}

			store := &TargetStore{
				nextID:  2,
				targets: []Target{original},
			}

			request := httptest.NewRequest(
				http.MethodPost,
				"/targets",
				strings.NewReader(tt.body),
			)
			request.Header.Set("Content-Type", "application/json")

			response := httptest.NewRecorder()

			store.createTarget(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf(
					"预期状态码 400，实际为 %d，正文：%s",
					response.Code,
					response.Body.String(),
				)
			}

			if len(store.targets) != 1 {
				t.Fatalf(
					"非法请求不应改变目标数量，实际为 %d",
					len(store.targets),
				)
			}

			if store.targets[0] != original {
				t.Errorf("非法请求修改了已有目标：%+v", store.targets[0])
			}

			if store.nextID != 2 {
				t.Errorf(
					"非法请求不应消耗 ID，实际 nextID 为 %d",
					store.nextID,
				)
			}
		})
	}
}
