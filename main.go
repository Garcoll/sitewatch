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
		fmt.Println("used: go run . <website>")
		return
	}
	target := os.Args[1]

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(target)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("检查失败：%s\n原因：%v\n", target, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("website: %s\n", target)
	fmt.Printf("HTTP status: %d\n", resp.StatusCode)
	fmt.Printf("response headers received in: %v\n", elapsed)
}
