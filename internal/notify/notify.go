package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var wxPusherAPI = "https://wxpusher.zjiecode.com/api/send/message"
const httpTimeout = 10 * time.Second

type Event struct {
	Provider string
	Success  bool
	Message  string
}

type Config struct {
	AppToken string
	UIDs     []string
}

type Notifier struct {
	cfg *Config
	cli *http.Client
}

func New(cfg *Config) *Notifier {
	return &Notifier{
		cfg: cfg,
		cli: &http.Client{Timeout: httpTimeout},
	}
}

func (n *Notifier) Enabled() bool {
	return n.cfg != nil && n.cfg.AppToken != "" && len(n.cfg.UIDs) > 0
}

func (n *Notifier) Send(event Event) error {
	if !n.Enabled() {
		return nil
	}

	content := fmt.Sprintf("SubMe 通知 [%s]\n%s", event.Provider, event.Message)

	body := map[string]interface{}{
		"appToken":    n.cfg.AppToken,
		"content":     content,
		"contentType": 1,
		"uids":        n.cfg.UIDs,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := n.cli.Post(wxPusherAPI, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("wxpusher request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wxpusher status: %d", resp.StatusCode)
	}

	return nil
}
