package services

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"qwen2api-go/upstream"
)

const QwenBaseURL = "https://chat.qwen.ai"

type UpstreamEvent struct {
	Type          string
	Phase         string
	Content       string
	ReasoningText string
	Status        string
	Extra         map[string]any
	Raw           map[string]any
}

type TokenVerifyResult struct {
	Valid      bool
	StatusCode string
	Message    string
}

func QwenHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json, text/event-stream")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
	headers.Set("x-request-id", QwenRequestID())
	headers.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	headers.Set("Referer", QwenBaseURL+"/")
	headers.Set("Origin", QwenBaseURL)
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
	for k, v := range BaxiaHeaders(token) {
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

func ParseQwenJSONEvent(data string) []UpstreamEvent {
	var obj map[string]any
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return nil
	}
	parsed := upstream.ParseQwenEvent(obj)
	out := make([]UpstreamEvent, 0, len(parsed))
	for _, evt := range parsed {
		out = append(out, UpstreamEvent{
			Type:          evt.Type,
			Phase:         evt.Phase,
			Content:       evt.Content,
			ReasoningText: evt.ReasoningText,
			Status:        evt.Status,
			Extra:         evt.Extra,
			Raw:           evt.Raw,
		})
	}
	return out
}

func FormatUpstreamError(obj map[string]any) string {
	if obj == nil {
		return ""
	}
	// Check for Baxia WAF captcha block: {"ret":["FAIL_SYS_USER_VALIDATE","RGV587_ERROR::SM::..."], "data":{...}}
	if retArr, ok := obj["ret"].([]any); ok && len(retArr) > 0 {
		for _, item := range retArr {
			if s, ok := item.(string); ok {
				if strings.HasPrefix(s, "FAIL_SYS_USER_VALIDATE") || strings.HasPrefix(s, "RGV587_ERROR") {
					return "Qwen upstream error code=baxia_captcha_block details=账号被Baxia验证码拦截，请稍后重试或更换账号"
				}
			}
		}
	}
	requestID := qwenFirstString(obj["request_id"], obj["response_id"])
	if requestID == "" {
		requestID = "-"
	}
	if success, ok := obj["success"].(bool); ok && !success {
		data, _ := obj["data"].(map[string]any)
		code := qwenFirstString(data["code"], obj["code"])
		if code == "" {
			code = "upstream_error"
		}
		details := qwenFirstString(data["details"], data["message"], obj["details"], obj["message"])
		return "Qwen upstream error code=" + code + " request_id=" + requestID + " details=" + details
	}
	if errObj, ok := obj["error"].(map[string]any); ok {
		code := qwenFirstString(errObj["code"])
		if code == "" {
			code = "upstream_error"
		}
		details := qwenFirstString(errObj["details"], errObj["message"], errObj["type"])
		return "Qwen upstream error code=" + code + " request_id=" + requestID + " details=" + details
	}
	if errText, ok := obj["error"].(string); ok && strings.TrimSpace(errText) != "" {
		return "Qwen upstream error request_id=" + requestID + " details=" + errText
	}
	return ""
}

func ExtractUpstreamError(text string) string {
	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "" || line == "[DONE]" || !strings.HasPrefix(line, "{") {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) != nil {
			continue
		}
		if message := FormatUpstreamError(obj); message != "" {
			return message
		}
	}
	return ""
}

func qwenFirstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
