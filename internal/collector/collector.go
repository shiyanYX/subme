package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second

type Config struct {
	ScriptDir  string // directory containing collector.js and _shared/
	ConfigPath string // path to config.yaml
	Proxy      string
}

func (c Config) ScriptPath() string {
	return "collector.js" // cmd.Dir is set to ScriptDir
}

type Result struct {
	Success             bool
	PanelURL            string
	SubscriptionURL     string
	SubscriptionContent []byte
	UpdateConfig        map[string]string
	ViaProxy            bool
	Error               string
	Stderr              string
}

type collectorOutput struct {
	Success             bool              `json:"success"`
	PanelURL            string            `json:"panel_url"`
	SubscriptionURL     string            `json:"subscription_url"`
	SubscriptionContent string            `json:"subscription_content,omitempty"`
	UpdateConfig        map[string]string `json:"update_config,omitempty"`
	ViaProxy            bool              `json:"via_proxy"`
	Error               string            `json:"error,omitempty"`
}

func Run(ctx context.Context, cfg Config) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", cfg.ScriptPath(), cfg.ConfigPath)
	cmd.Dir = cfg.ScriptDir

	if cfg.Proxy != "" {
		if strings.HasPrefix(cfg.Proxy, "socks5://") {
			cmd.Env = append(os.Environ(), "SOCKS_PROXY="+cfg.Proxy)
		} else {
			cmd.Env = append(os.Environ(), "HTTP_PROXY="+cfg.Proxy, "HTTPS_PROXY="+cfg.Proxy)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &Result{Error: "collector timed out", Stderr: stderr.String()}, nil
		}
		return &Result{Error: fmt.Sprintf("collector process error: %v", err), Stderr: stderr.String()}, nil
	}

	var out collectorOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return &Result{
			Error:  fmt.Sprintf("invalid collector output: %v", err),
			Stderr: stderr.String(),
		}, nil
	}

	result := &Result{
		Success:             out.Success,
		PanelURL:            out.PanelURL,
		SubscriptionURL:     out.SubscriptionURL,
		SubscriptionContent: []byte(out.SubscriptionContent),
		UpdateConfig:        out.UpdateConfig,
		ViaProxy:            out.ViaProxy,
		Error:               out.Error,
		Stderr:              stderr.String(),
	}

	return result, nil
}


