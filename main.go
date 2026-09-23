package main

import (
	"flag"
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
	var concurrency int
	var timeout time.Duration

	flag.IntVar(&concurrency, "concurrency", 3, "最多同时检查的网址数量")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "每个请求的超时时间，例如 500ms、3s")
	flag.Parse()

	if concurrency < 1 {
		fmt.Fprintln(os.Stderr, "参数错误: -concurrency 必须大于0")
		os.Exit(2)
	}

	if timeout <= 0 {
		fmt.Fprintln(os.Stderr, "参数错误：-timeout 必须大于 0,例如 500ms、3s")
		os.Exit(2)
	}

	targets := flag.Args()

	if len(targets) == 0 {
		fmt.Println("used: go run . [-concurrency count]<web1>[web2...]")
		return
	}

	client := &http.Client{
		Timeout: timeout,
	}

	batchStart := time.Now()
	results := checkWebsites(client, targets, concurrency)
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
