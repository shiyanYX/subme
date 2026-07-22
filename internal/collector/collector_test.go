package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeMockCollector(t *testing.T, dir, script string) {
	t.Helper()
	jsPath := filepath.Join(dir, "collector.js")
	if err := os.WriteFile(jsPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

func writeMockConfig(t *testing.T, dir string, content string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func makeCfg(t *testing.T, script string, configYAML string) Config {
	t.Helper()
	dir := t.TempDir()
	writeMockCollector(t, dir, script)
	cfgPath := writeMockConfig(t, dir, configYAML)
	return Config{
		ScriptDir:  dir,
		ConfigPath: cfgPath,
	}
}

func TestRun_Success(t *testing.T) {
	script := `process.stdout.write(JSON.stringify({success:true, panel_url:"https://panel.example.com", subscription_url:"https://sub.example.com/clash", update_config:{panel_url:"https://new.example.com"}}));`

	cfg := makeCfg(t, script, "clash_name: test\n")
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.PanelURL != "https://panel.example.com" {
		t.Errorf("PanelURL: got %q", result.PanelURL)
	}
	if result.SubscriptionURL != "https://sub.example.com/clash" {
		t.Errorf("SubscriptionURL: got %q", result.SubscriptionURL)
	}
}

func TestRun_Failure(t *testing.T) {
	script := `process.stdout.write(JSON.stringify({success:false, error:"login failed"}));`

	cfg := makeCfg(t, script, "clash_name: test\n")
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Error("expected success=false")
	}
	if result.Error != "login failed" {
		t.Errorf("Error: got %q", result.Error)
	}
}

func TestRun_InvalidJSON(t *testing.T) {
	script := "process.stdout.write('this is not json');"
	cfg := makeCfg(t, script, "clash_name: test\n")
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Error("expected success=false for invalid JSON")
	}
	if result.Error == "" {
		t.Error("expected error message about invalid output")
	}
}

func TestRun_ProcessError(t *testing.T) {
	cfg := Config{
		ScriptDir:  t.TempDir(),
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
	}
	// No collector.js in ScriptDir — node will fail
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Error("expected success=false")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestRun_Timeout(t *testing.T) {
	script := `
const now = Date.now();
while (Date.now() - now < 60000) {} // infinite loop for 60s
process.stdout.write('{}');
`
	cfg := makeCfg(t, script, "clash_name: test\n")
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Error("expected success=false on timeout")
	}
	if result.Error != "collector timed out" {
		t.Errorf("expected timeout error, got %q", result.Error)
	}
}

func TestRun_WithProxy(t *testing.T) {
	script := "const e = process.env; process.stdout.write(JSON.stringify({success:true, subscription_url: 'https://sub.example.com', proxy_used: (e.HTTP_PROXY || '')}));"
	cfg := makeCfg(t, script, "clash_name: test\n")
	cfg.Proxy = "http://proxy.example.com:8080"
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Errorf("expected success=true, error=%q", result.Error)
	}
}

func TestScriptPath(t *testing.T) {
	cfg := Config{ScriptDir: "/some/dir"}
	if got := cfg.ScriptPath(); got != "collector.js" {
		t.Errorf("ScriptPath: got %q, want %q", got, "collector.js")
	}
}

func TestRun_StderrCapture(t *testing.T) {
	script := "console.error('debug info'); process.stdout.write(JSON.stringify({success:true, subscription_url:'https://sub.example.com'}));"
	cfg := makeCfg(t, script, "clash_name: test\n")
	result, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stderr == "" {
		t.Error("expected stderr to be captured")
	}
}
