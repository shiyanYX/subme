package cache

import (
	"testing"
)

func TestContent(t *testing.T) {
	e := Entry{
		YAML:    []byte("yaml-body"),
		Base64:  []byte("base64-body"),
		SingBox: []byte("json-body"),
	}
	cases := []struct {
		format string
		want   string
	}{
		{"clash", "yaml-body"},
		{"v2ray", "base64-body"},
		{"singbox", "json-body"},
		{"unknown", "yaml-body"},
	}
	for _, c := range cases {
		if got := string(e.Content(c.format)); got != c.want {
			t.Fatalf("Content(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}

func TestContentFallsBackToYAML(t *testing.T) {
	e := Entry{YAML: []byte("only-yaml")}
	if got := string(e.Content("v2ray")); got != "only-yaml" {
		t.Fatalf("Content(v2ray) = %q, want fallback %q", got, "only-yaml")
	}
	if got := string(e.Content("singbox")); got != "only-yaml" {
		t.Fatalf("Content(singbox) = %q, want fallback %q", got, "only-yaml")
	}
}

func TestSetAndGetRoundtrip(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := Formats{Clash: []byte("y"), V2Ray: []byte("b64"), SingBox: []byte("j")}
	if err := c.Set("prov-a", f, nil); err != nil {
		t.Fatal(err)
	}
	e, err := c.Get("prov-a")
	if err != nil {
		t.Fatal(err)
	}
	if string(e.YAML) != "y" || string(e.Base64) != "b64" || string(e.SingBox) != "j" {
		t.Fatalf("roundtrip mismatch: yaml=%q base64=%q singbox=%q", e.YAML, e.Base64, e.SingBox)
	}
}
