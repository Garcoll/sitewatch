package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// CheckResult 保存一次检查的结果。
type CheckResult struct {
	URL        string
	StatusCode int
	Duration   time.Duration
	Err        error
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法：go run . <网址1> [网址2 ...]")
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	results := make([]CheckResult, 0, len(os.Args)-1)

	for _, target := range os.Args[1:] {
		result := checkWebsite(client, target)
		results = append(results, result)
	}

	normal := 0
	abnormal := 0
	failed := 0

	for _, result := range results {
		if result.Err != nil {
			failed++
			fmt.Printf(
				"[请求失败] %s | %v | 耗时 %v\n",
				result.URL, result.Err, result.Duration,
			)
			continue
		}

		if result.StatusCode >= 200 && result.StatusCode < 400 {
			normal++
			fmt.Printf(
				"[状态正常] %s | HTTP %d | 耗时 %v\n",
				result.URL, result.StatusCode, result.Duration,
			)
		} else {
			abnormal++
			fmt.Printf(
				"[状态异常] %s | HTTP %d | 耗时 %v\n",
				result.URL, result.StatusCode, result.Duration,
			)
		}
	}

	fmt.Printf(
		"\n共检查 %d 个网址：正常 %d，状态异常 %d，请求失败 %d\n",
		len(results), normal, abnormal, failed,
	)
}

func checkWebsite(client *http.Client, target string) CheckResult {
	start := time.Now()
	resp, err := client.Get(target)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:      target,
			Duration: duration,
			Err:      err,
		}
	}
	defer resp.Body.Close()

	return CheckResult{
		URL:        target,
		StatusCode: resp.StatusCode,
		Duration:   duration,
	}
}
