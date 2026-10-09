package checker

import (
	"net/http"
	"sort"
	"sync"
	"time"
)

// CheckResult 保存一次检查的结果。
type CheckResult struct {
	Index      int
	URL        string
	StatusCode int
	Duration   time.Duration
	Err        error
}

type checkJob struct {
	Index  int
	Target string
}

// CheckWebsite 检查一个网址并记录响应状态、耗时或请求错误。
func CheckWebsite(client *http.Client, target string) CheckResult {
	start := time.Now()
	resp, err := client.Get(target)
	duration := time.Since(start)
	if err != nil {
		return CheckResult{URL: target, Duration: duration, Err: err}
	}
	defer resp.Body.Close()
	return CheckResult{URL: target, StatusCode: resp.StatusCode, Duration: duration}
}

// CheckWebsites 并发检查一批网址，并按输入顺序返回结果。
func CheckWebsites(client *http.Client, targets []string, concurrency int) []CheckResult {
	if len(targets) == 0 {
		return nil
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(targets) {
		concurrency = len(targets)
	}

	jobs := make(chan checkJob)
	results := make(chan CheckResult)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := CheckWebsite(client, job.Target)
				result.Index = job.Index
				results <- result
			}
		}()
	}
	go func() {
		for index, target := range targets {
			jobs <- checkJob{Index: index, Target: target}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make([]CheckResult, 0, len(targets))
	for result := range results {
		collected = append(collected, result)
	}
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Index < collected[j].Index
	})
	return collected
}
