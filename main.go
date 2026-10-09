package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"example.com/sitewatch/internal/checker"
)

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
	results := checker.CheckWebsites(client, targets, concurrency)
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
