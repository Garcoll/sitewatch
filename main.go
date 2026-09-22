package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
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

	batchStart := time.Now()
	results := checkWebsites(client, os.Args[1:], 3)
	batchDuration := time.Since(batchStart)

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

	fmt.Printf("整批检查耗时：%v\n", batchDuration)
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

func checkWebsites(
	client *http.Client,
	targets []string,
	concurrency int,
) []CheckResult {
	if len(targets) == 0 {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(targets) {
		concurrency = len(targets)
	}

	jobs := make(chan string)
	results := make(chan CheckResult)

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for target := range jobs {
				results <- checkWebsite(client, target)
			}
		}()
	}

	// 独立发送任务，让调用方可以同时接收结果。
	go func() {
		for _, target := range targets {
			jobs <- target
		}
		close(jobs)
	}()

	// 所有 worker 停止发送后，才能关闭结果通道。
	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make([]CheckResult, 0, len(targets))
	for result := range results {
		collected = append(collected, result)
	}

	return collected
}
