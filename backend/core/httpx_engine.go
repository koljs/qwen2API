package core

import (
	cryptorand "crypto/rand"
	"fmt"
	"net/http"
	"time"

	"qwen2api-go/services"
)

func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 1800 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func QwenHeaders(token string, cookies string) http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json, text/event-stream")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
	headers.Set("x-request-id", QwenRequestID())
	headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	headers.Set("Referer", "https://chat.qwen.ai/")
	headers.Set("Origin", "https://chat.qwen.ai")
	headers.Set("Connection", "keep-alive")
	headers.Set("sec-ch-ua", `"Google Chrome";v="149", "Chromium";v="149", "Not)A;Brand";v="24"`)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", `"Windows"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	if token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	// Baxia WAF headers
	for k, v := range services.BaxiaHeaders(token, cookies) {
		headers.Set(k, v)
	}
	return headers
}

func QwenRequestID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uint32(now), uint16(now>>32), uint16(now>>48), uint16(now>>16), uint64(now))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
