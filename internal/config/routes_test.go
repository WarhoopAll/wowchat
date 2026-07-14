package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoutesAndFiltersParsing(t *testing.T) {
	for _, kv := range os.Environ() {
		for _, pfx := range []string{"WOW_", "DISCORD_", "FILTER_", "CHAT_"} {
			if hasPrefix(kv, pfx) {
				eq := indexByte(kv, '=')
				if eq > 0 {
					os.Unsetenv(kv[:eq])
				}
			}
		}
	}

	body := `
WOW_PLATFORM=Mac
WOW_VERSION=3.3.5
WOW_REALMLIST_HOST=127.0.0.1
WOW_REALMLIST_PORT=3724
WOW_REALM=Trinity
WOW_ACCOUNT=a
WOW_PASSWORD=p
WOW_CHARACTER=c

FILTERS_ENABLED=true
FILTER_PATTERNS=.*spam.* .*gold.*

CHAT_ROUTE_COUNT=3
CHAT_ROUTE_1_DIRECTION=both
CHAT_ROUTE_1_WOW_TYPE=Channel
CHAT_ROUTE_1_WOW_CHANNEL=LookingForGroup
CHAT_ROUTE_1_WOW_FORMAT=[%user]: %message
CHAT_ROUTE_1_DISCORD_CHANNEL=123456789
CHAT_ROUTE_1_DISCORD_FORMAT=[%target] [%user]: %message

CHAT_ROUTE_2_DIRECTION=discord_to_wow
CHAT_ROUTE_2_WOW_TYPE=Guild
CHAT_ROUTE_2_WOW_FORMAT=[%user]: %message
CHAT_ROUTE_2_DISCORD_CHANNEL=guild-chat
CHAT_ROUTE_2_DISCORD_FORMAT=[%user]: %message

CHAT_ROUTE_3_DIRECTION=wow_to_discord
CHAT_ROUTE_3_WOW_TYPE=System
CHAT_ROUTE_3_DISCORD_FORMAT=[SYSTEM]: %message
CHAT_ROUTE_3_DISCORD_CHANNEL=123456789
`
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Routes) != 3 {
		t.Fatalf("routes = %d, want 3", len(cfg.Routes))
	}

	r1 := cfg.Routes[0]
	if r1.Direction != "both" || r1.WowType != ChatTypeChannel || r1.WowChannel != "LookingForGroup" {
		t.Errorf("route1 wrong: %+v", r1)
	}
	if r1.DiscordChan != "123456789" || r1.DiscordFmt != "[%target] [%user]: %message" {
		t.Errorf("route1 discord: %+v", r1)
	}

	r2 := cfg.Routes[1]
	if r2.Direction != "discord_to_wow" || r2.WowType != ChatTypeGuild || r2.WowChannel != "" {
		t.Errorf("route2 wrong: %+v", r2)
	}

	if cfg.Routes[2].WowType != ChatTypeSystem {
		t.Errorf("route3 type = %#x, want System", cfg.Routes[2].WowType)
	}

	wantCh := []string{"LookingForGroup"}
	if len(cfg.Channels) != 1 || cfg.Channels[0] != "LookingForGroup" {
		t.Errorf("channels = %#v, want %#v", cfg.Channels, wantCh)
	}

	t.Logf("FiltersEnabled=%v FilterPatterns=%q", cfg.FiltersEnabled, cfg.FilterPatterns)
	if len(cfg.FilterPatterns) != 2 {
		t.Errorf("filter patterns = %v, want 2", cfg.FilterPatterns)
	}
}

func hasPrefix(s, pfx string) bool {
	if len(s) < len(pfx) {
		return false
	}
	return s[:len(pfx)] == pfx
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func TestParseChatType(t *testing.T) {
	cases := map[string]byte{
		"channel": ChatTypeChannel,
		"Channel": ChatTypeChannel,
		"CUSTOM":  ChatTypeChannel,
		"guild":   ChatTypeGuild,
		"system":  ChatTypeSystem,
		"say":     ChatTypeSay,
		"yell":    ChatTypeYell,
		"emote":   ChatTypeEmote,
		"whisper": ChatTypeWhisper,
		"officer": ChatTypeOfficer,
		"bogus":   0xFF,
		"":        0xFF,
	}
	for in, want := range cases {
		if got := ParseChatType(in); got != want {
			t.Errorf("ParseChatType(%q) = %#x, want %#x", in, got, want)
		}
	}
}
