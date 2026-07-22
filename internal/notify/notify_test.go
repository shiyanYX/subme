package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{"nil config", nil, false},
		{"empty config", &Config{}, false},
		{"no uids", &Config{AppToken: "token"}, false},
		{"no app token", &Config{UIDs: []string{"uid1"}}, false},
		{"fully configured", &Config{AppToken: "token", UIDs: []string{"uid1"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := New(tt.cfg)
			if got := n.Enabled(); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSend_NotEnabled(t *testing.T) {
	n := New(nil)
	err := n.Send(Event{Provider: "test", Success: true, Message: "ok"})
	if err != nil {
		t.Fatalf("expected no error when disabled, got %v", err)
	}
}

func TestSend_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origAPI := wxPusherAPI
	wxPusherAPI = server.URL
	defer func() { wxPusherAPI = origAPI }()

	n := New(&Config{AppToken: "test-token", UIDs: []string{"uid1", "uid2"}})
	err := n.Send(Event{
		Provider: "MyProvider",
		Success:  true,
		Message:  "Collection succeeded",
	})
	if err != nil {
		t.Fatal(err)
	}

	if receivedBody == nil {
		t.Fatal("server did not receive a request body")
	}

	if receivedBody["appToken"] != "test-token" {
		t.Errorf("appToken: got %v", receivedBody["appToken"])
	}

	uids, ok := receivedBody["uids"].([]interface{})
	if !ok || len(uids) != 2 {
		t.Fatalf("uids: expected 2 items, got %v", receivedBody["uids"])
	}
	if uids[0] != "uid1" || uids[1] != "uid2" {
		t.Errorf("uids: got %v", uids)
	}

	content, _ := receivedBody["content"].(string)
	if content == "" {
		t.Error("content should not be empty")
	}
	if receivedBody["contentType"] != float64(1) {
		t.Errorf("contentType: got %v", receivedBody["contentType"])
	}
}

func TestSend_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origAPI := wxPusherAPI
	wxPusherAPI = server.URL
	defer func() { wxPusherAPI = origAPI }()

	n := New(&Config{AppToken: "token", UIDs: []string{"uid"}})
	err := n.Send(Event{Provider: "p", Message: "m"})
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestSend_ContainsProviderAndMessage(t *testing.T) {
	var content string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		content, _ = body["content"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origAPI := wxPusherAPI
	wxPusherAPI = server.URL
	defer func() { wxPusherAPI = origAPI }()

	n := New(&Config{AppToken: "t", UIDs: []string{"u"}})
	n.Send(Event{Provider: "MyProvider", Success: false, Message: "Login failed"})

	if content == "" {
		t.Fatal("content was empty")
	}
}
