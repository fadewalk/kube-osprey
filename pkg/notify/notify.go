// Package notify delivers notification messages to outbound webhook-style
// channels (DingTalk, WeCom, Feishu, Slack or a generic JSON receiver).
package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zxh326/kite/pkg/model"
)

const httpClientTimeout = 10 * time.Second

var client = &http.Client{Timeout: httpClientTimeout}

// Message is what a channel receives, formatted per channel type on send.
type Message struct {
	Title string
	Text  string
}

// Send posts the message to the channel endpoint, applying the
// channel-type-specific payload format and signing scheme.
func Send(channel model.NotificationChannel, msg Message) error {
	payload, endpoint, err := formatPayload(channel, msg)
	if err != nil {
		return err
	}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}

func formatPayload(channel model.NotificationChannel, msg Message) ([]byte, string, error) {
	channelType := strings.ToLower(channel.Type)
	secret := channel.Secret
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		text = msg.Title
	}

	var body any
	switch channelType {
	case "dingtalk":
		body = map[string]any{
			"msgtype":  "markdown",
			"markdown": map[string]string{"title": msg.Title, "text": text},
		}
	case "wecom":
		// WeCom bot content is capped at 4096 bytes.
		content := text
		if len(content) > 4000 {
			content = content[:4000]
		}
		body = map[string]any{
			"msgtype":  "markdown",
			"markdown": map[string]string{"content": content},
		}
	case "feishu":
		body = map[string]any{
			"msg_type": "text",
			"content":  map[string]string{"text": text},
		}
	case "slack":
		body = map[string]any{"text": text}
	case "webhook":
		body = map[string]any{"title": msg.Title, "text": text}
	default:
		return nil, "", fmt.Errorf("unknown notification channel type %q", channelType)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	endpoint := channel.Endpoint

	// Outbound signing schemes, applied after the payload is built.
	switch channelType {
	case "dingtalk":
		if secret != "" {
			timestamp := fmt.Sprint(time.Now().UnixMilli())
			sign := signDingTalk(timestamp, string(secret))
			sep := "?"
			if strings.Contains(endpoint, "?") {
				sep = "&"
			}
			endpoint += sep + "timestamp=" + url.QueryEscape(timestamp) + "&sign=" + url.QueryEscape(sign)
		}
	case "feishu":
		if secret != "" {
			timestamp := fmt.Sprint(time.Now().Unix())
			// Feishu: sign = base64(HMAC-SHA256(key = timestamp+"\n"+secret, data = ""))
			mac := hmac.New(sha256.New, []byte(timestamp+"\n"+string(secret)))
			mac.Write([]byte(""))
			sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
			bodyMap, _ := json.Marshal(map[string]any{
				"msg_type":  "text",
				"content":   map[string]string{"text": text},
				"timestamp": timestamp,
				"sign":      sign,
			})
			encoded = bodyMap
		}
	}
	return encoded, endpoint, nil
}

func signDingTalk(timestamp, secret string) string {
	// DingTalk: sign = base64(HMAC-SHA256(key = secret, data = timestamp+"\n"+secret))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "\n" + secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
