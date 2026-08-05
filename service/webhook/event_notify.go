package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// WebhookEvent 事件类型
type EventType string

const (
	EventRequestComplete  EventType = "request.complete"
	EventRequestFailed    EventType = "request.failed"
	EventQuotaLow         EventType = "quota.low"
	EventQuotaExhausted   EventType = "quota.exhausted"
	EventChannelDown      EventType = "channel.down"
	EventChannelRecovered EventType = "channel.recovered"
	EventUserRegister     EventType = "user.registered"
	EventPaymentReceived  EventType = "payment.received"
)

// WebhookEvent 事件结构
type WebhookEvent struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// webhookURLs 用户配置的 Webhook URL（从 options 读取）
func getWebhookURLs() []string {
	// 从系统配置读取
	urlStr := common.OptionMap["WebhookURLs"]
	if urlStr == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(urlStr), &urls); err != nil {
		return nil
	}
	return urls
}

// Notify 发送事件通知到所有配置的 Webhook URL
func Notify(eventType EventType, data interface{}) {
	urls := getWebhookURLs()
	if len(urls) == 0 {
		return
	}

	event := WebhookEvent{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		common.SysError("webhook marshal error: " + err.Error())
		return
	}

	for _, url := range urls {
		go sendWebhook(url, payload)
	}
}

func sendWebhook(url string, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "YuxinAPI-Webhook/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		common.SysError(fmt.Sprintf("webhook send error to %s: %s", url, err.Error()))
		return
	}
	resp.Body.Close()
}

// NotifyQuotaLow 余额不足通知
func NotifyQuotaLow(userID int, username string, remaining int64) {
	Notify(EventQuotaLow, map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"remaining": remaining,
		"message":   fmt.Sprintf("用户 %s 余额即将耗尽，剩余: %d", username, remaining),
	})
}

// NotifyChannelDown 渠道故障通知
func NotifyChannelDown(channelID int, channelName string, reason string) {
	Notify(EventChannelDown, map[string]interface{}{
		"channel_id":   channelID,
		"channel_name": channelName,
		"reason":       reason,
		"message":      fmt.Sprintf("渠道 %s (#%d) 发生故障: %s", channelName, channelID, reason),
	})
}
