package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	url := "http://10.161.30.88:8071/cmsdebugging/showversion?debugging=89711f468e1ed47d1ddc698c108cbc7e0e23dd3c84bb1743cc664b656de4578f"

	body, err := fetchURL(url)
	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	fmt.Println("响应内容:")
	fmt.Println(body)
}

// fetchURL 发送 HTTP GET 请求并返回响应 body（字符串）
func fetchURL(url string) (string, error) {
	// 创建带超时的 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second, // 整个请求超时时间
	}

	// 使用 context 实现更精细的控制（可选）
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 可选：设置 User-Agent 等头部
	req.Header.Set("User-Agent", "curl/8.5.0")
	//req.Header.Set("Accept-Encoding", "*")
	//req.Header.Set("Accept", "*/*")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求执行失败: %w", err)
	}
	defer resp.Body.Close() // ✅ 必须关闭 Body

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 读取 body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	return string(bodyBytes), nil
}
