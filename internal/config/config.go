package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	ClashName   string `yaml:"clash_name"`
	Interval    int    `yaml:"interval"`
	PanelURL    string `yaml:"panel_url"`
	LandingPage string `yaml:"landing_page"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}

type WxPusherConfig struct {
	AppToken string   `json:"app_token" yaml:"app_token"`
	UIDs     []string `json:"uids" yaml:"uids"`
}

type NotifyOn struct {
	CollectFailure bool `json:"collect_failure" yaml:"collect_failure"`
	RefreshFailure bool `json:"refresh_failure" yaml:"refresh_failure"`
}

type SystemSettings struct {
	Port            int            `json:"port" yaml:"port"`
	RefreshInterval int            `json:"refresh_interval" yaml:"refresh_interval"`
	Proxy           string         `json:"proxy" yaml:"proxy"`
	WxPusher        WxPusherConfig `json:"wxpusher" yaml:"wxpusher"`
	NotifyOn        NotifyOn       `json:"notify_on" yaml:"notify_on"`
}

func LoadProviderConfig(path string) (*ProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider config: %w", err)
	}
	var pc ProviderConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parse provider config: %w", err)
	}
	if pc.ClashName == "" {
		return nil, fmt.Errorf("clash_name is required")
	}
	if pc.Interval <= 0 {
		pc.Interval = 3600
	}
	return &pc, nil
}

func SaveProviderConfig(path string, pc *ProviderConfig) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshal provider config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func UpdateProviderConfig(path string, updates map[string]string) error {
	pc, err := LoadProviderConfig(path)
	if err != nil {
		return err
	}
	for k, v := range updates {
		switch k {
		case "panel_url":
			pc.PanelURL = v
		case "landing_page":
			pc.LandingPage = v
		case "username":
			pc.Username = v
		case "password":
			pc.Password = v
		}
	}
	return SaveProviderConfig(path, pc)
}
