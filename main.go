package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	//target := "http://www.baidu.com"
	if len(os.Args) < 2 {
		fmt.Println("用法：go run . <网址1> [网址2 ...]")
		return
	}
	for _, target := range os.Args[1:] {
		checkWebsite(target)
	}

}

func checkWebsite(target string) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(target)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("检查失败：%s\n原因：%v\n", target, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("website: %s\n", target)
	fmt.Printf("HTTP status: %d\n", resp.StatusCode)
	fmt.Printf("response headers received in: %v\n", elapsed)
}
