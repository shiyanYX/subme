package server

import (
	"net/http/httptest"
	"testing"
)

func TestSubscriptionFormatFor(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		path string
		want string
	}{
		{"clash-verge", "clash-verge/2.4.0", "", "clash"},
		{"clash meta uppercase", "ClashMetaForAndroid/2.8.0", "", "clash"},
		{"sing-box", "sing-box/1.9.0", "", "singbox"},
		{"v2rayN", "v2rayN/7.24.2", "", "v2ray"},
		{"nekobox", "NekoBox/1.0", "", "v2ray"},
		{"subme collector", "SubMe/1.0", "", "v2ray"},
		{"empty ua", "", "", "v2ray"},
		{"format param overrides", "v2rayN/7.24.2", "?format=clash", "clash"},
		{"format singbox", "v2rayN/7.24.2", "?format=sing-box", "singbox"},
		{"format base64", "clash-verge/2.4.0", "?format=base64", "v2ray"},
		{"invalid format falls through", "v2rayN/7.24.2", "?format=bogus", "v2ray"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/sub/x"+c.path, nil)
			r.Header.Set("User-Agent", c.ua)
			if got := subscriptionFormatFor(r); got != c.want {
				t.Fatalf("subscriptionFormatFor(ua=%q path=%q) = %q, want %q", c.ua, c.path, got, c.want)
			}
		})
	}
}
