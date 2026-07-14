package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeEnv(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestLoadValidEnv(t *testing.T) {
	body := `
WOW_PLATFORM=Mac
WOW_VERSION=3.3.5
WOW_REALMLIST_HOST=127.0.0.1
WOW_REALMLIST_PORT=3724
WOW_REALM=Trinity
WOW_ACCOUNT=testacct
WOW_PASSWORD=secret
WOW_CHARACTER=Pal

DISCORD_TOKEN=tok
DISCORD_ENABLE_DOT_COMMANDS=true
DISCORD_DOT_COMMANDS_WHITELIST=server info

CHAT_ROUTE_COUNT=3
CHAT_ROUTE_1_DIRECTION=wow_to_discord
CHAT_ROUTE_1_WOW_TYPE=Channel
CHAT_ROUTE_1_WOW_CHANNEL=LookingForGroup
CHAT_ROUTE_2_DIRECTION=wow_to_discord
CHAT_ROUTE_2_WOW_TYPE=Channel
CHAT_ROUTE_2_WOW_CHANNEL=Поиск спутников
CHAT_ROUTE_3_DIRECTION=wow_to_discord
CHAT_ROUTE_3_WOW_TYPE=System
`
	p := writeEnv(t, ".env", body)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WoW.Build != 12340 {
		t.Errorf("build = %d, want 12340", cfg.WoW.Build)
	}
	if cfg.WoW.Platform != PlatformMac {
		t.Errorf("platform = %q, want Mac", cfg.WoW.Platform)
	}
	if cfg.WoW.Account != "TESTACCT" {
		t.Errorf("account = %q, want TESTACCT", cfg.WoW.Account)
	}
	want := []string{"LookingForGroup", "Поиск спутников"}
	if !reflect.DeepEqual(cfg.Channels, want) {
		t.Errorf("channels = %#v, want %#v", cfg.Channels, want)
	}
}

func TestMissingRequiredProducesErrors(t *testing.T) {
	p := writeEnv(t, ".env", `WOW_VERSION=3.3.5`+"\n")
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for missing required vars")
	}
	for _, key := range []string{"WOW_ACCOUNT", "WOW_PASSWORD", "WOW_CHARACTER", "WOW_REALMLIST_HOST", "WOW_REALM"} {
		if !contains(err.Error(), key) {
			t.Errorf("error %q missing key %q", err.Error(), key)
		}
	}
}

func TestProcessEnvOverridesDotEnv(t *testing.T) {
	body := `WOW_ACCOUNT=fromfile
WOW_PASSWORD=p
WOW_CHARACTER=c
WOW_REALMLIST_HOST=127.0.0.1
WOW_REALMLIST_PORT=3724
WOW_REALM=Trinity
WOW_VERSION=3.3.5
`
	p := writeEnv(t, ".env", body)
	t.Setenv("WOW_ACCOUNT", "fromproc")
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WoW.Account != "FROMPROC" {
		t.Errorf("account = %q, want FROMPROC (process env must win)", cfg.WoW.Account)
	}
}

func TestConvertToUpperAsciiOnly(t *testing.T) {
	got := convertToUpper("abcДЕ123")
	want := "ABCДЕ123"
	if got != want {
		t.Errorf("convertToUpper = %q, want %q", got, want)
	}
}

func TestVarExpansion(t *testing.T) {
	body := `DISCORD_CHANNEL_LFG=12345
CHAT_ROUTE_1_DISCORD_CHANNEL=${DISCORD_CHANNEL_LFG}
`
	values, err := readDotEnv(writeEnv(t, ".env", body))
	if err != nil {
		t.Fatalf("readDotEnv: %v", err)
	}
	if values["CHAT_ROUTE_1_DISCORD_CHANNEL"] != "12345" {
		t.Errorf("var expansion = %q, want 12345", values["CHAT_ROUTE_1_DISCORD_CHANNEL"])
	}
}

func TestBuildForVersionUnsupported(t *testing.T) {
	if _, err := buildForVersion("9.9.9"); err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestProxyConfig(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
		discord bool
		realm   bool
		host    string
	}{
		{
			name:    "disabled by default",
			body:    baseEnv(),
			wantErr: false,
		},
		{
			name: "valid http proxy both connects",
			body: baseEnv() + `
PROXY_URL=http://user:pass@proxy.example.com:3128
PROXY_DISCORD_CONNECT=true
PROXY_REALM_CONNECT=true
`,
			discord: true,
			realm:   true,
			host:    "proxy.example.com:3128",
		},
		{
			name: "https scheme accepted",
			body: baseEnv() + `
PROXY_URL=https://proxy.example.com:3128
PROXY_REALM_CONNECT=1
`,
			realm: true,
			host:  "proxy.example.com:3128",
		},
		{
			name: "socks5 scheme accepted",
			body: baseEnv() + `
PROXY_URL=socks5://user:pass@proxy.example.com:1080
PROXY_DISCORD_CONNECT=true
`,
			discord: true,
			host:    "proxy.example.com:1080",
		},
		{
			name: "malformed rejected",
			body: baseEnv() + `
PROXY_URL=http://
`,
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeEnv(t, ".env", c.body)
			cfg, err := Load(p)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (host=%v)", cfg.ProxyURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.discord != cfg.Proxy.DiscordConnect {
				t.Errorf("DiscordConnect = %v, want %v", cfg.Proxy.DiscordConnect, c.discord)
			}
			if c.realm != cfg.Proxy.RealmConnect {
				t.Errorf("RealmConnect = %v, want %v", cfg.Proxy.RealmConnect, c.realm)
			}
			if c.host == "" {
				if cfg.ProxyURL != nil {
					t.Errorf("ProxyURL = %v, want nil", cfg.ProxyURL)
				}
				return
			}
			if cfg.ProxyURL == nil {
				t.Fatalf("ProxyURL nil, want host %q", c.host)
			}
			got := cfg.ProxyURL.Host
			if got != c.host {
				t.Errorf("ProxyURL.Host = %q, want %q", got, c.host)
			}
		})
	}
}

func baseEnv() string {
	return `WOW_PLATFORM=Mac
WOW_VERSION=3.3.5
WOW_REALMLIST_HOST=127.0.0.1
WOW_REALMLIST_PORT=3724
WOW_REALM=Trinity
WOW_ACCOUNT=testacct
WOW_PASSWORD=secret
WOW_CHARACTER=Pal
DISCORD_TOKEN=tok
CHAT_ROUTE_COUNT=1
CHAT_ROUTE_1_DIRECTION=wow_to_discord
CHAT_ROUTE_1_WOW_TYPE=System
`
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
