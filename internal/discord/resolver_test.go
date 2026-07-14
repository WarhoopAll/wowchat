package discord

import "testing"

func TestStripColorCoding(t *testing.T) {
	cases := map[string]string{
		"|cffff0000hello|r":                    "hello",
		"plain":                                "plain",
		"|cff00ff00green|r and |cffff0000red|r": "green and red",
		"|cffabcdef":                           "",
	}
	for in, want := range cases {
		if got := stripColorCoding(in); got != want {
			t.Errorf("stripColorCoding(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveLinks(t *testing.T) {
	r := newResolver("")
	if r.linkSite != defaultLinkSite {
		t.Fatalf("default link site = %q", r.linkSite)
	}

	item := "|cffffffff|Hitem:12345:0:0:0:0:0:0:0:0|h[Thunderfury]|h|r"
	got := r.resolveLinks(item)
	want := "[[Thunderfury]](" + defaultLinkSite + "?item=12345)"
	if got != want {
		t.Errorf("resolveLinks item = %q, want %q", got, want)
	}

	ach := "|cffffff00|Hachievement:6:0:0:0:0:0:0:0|h[Level 10]|h|r"
	got = r.resolveLinks(ach)
	want = "[[Level 10]](" + defaultLinkSite + "?achievement=6)"
	if got != want {
		t.Errorf("resolveLinks achievement = %q, want %q", got, want)
	}
}

func TestResolveLinksCustomSite(t *testing.T) {
	r := newResolver("http://example.com")
	got := r.resolveLinks("|cff71d5ff|Hspell:133:0|h[Fireball]|h|r")
	want := "[[Fireball]](http://example.com?spell=133)"
	if got != want {
		t.Errorf("resolveLinks spell = %q, want %q", got, want)
	}
}

func TestStripEmoji(t *testing.T) {
	cases := map[string]string{
		"hello 😀 world":           "hello  world",
		"gg <:pog:1234567890>":    "gg ",
		"wave <a:wave:987654321>": "wave ",
		"❤️🔥 text":                " text",
		"обычный текст":            "обычный текст",
		"⭐💀❌":                    "",
	}
	for in, want := range cases {
		if got := stripEmoji(in); got != want {
			t.Errorf("stripEmoji(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveRaidIcons(t *testing.T) {
	r := newResolver("")
	cases := map[string]string{
		"pull {skull} now":     "pull 💀 now",
		"{star}{moon}":         "⭐🌙",
		"{звезда}{полумесяц}": "⭐🌙",
		"{cross}{circle}":      "❌⚫",
		"{крест}{круг}":        "❌⚫",
		"{diamond}{square}":    "♦️◼️",
		"{ромб}{квадрат}":      "♦️◼️",
		"{triangle}":           "🔺",
		"{треугольник}":        "🔺",
		"no icons here":        "no icons here",
	}
	for in, want := range cases {
		if got := r.resolveRaidIcons(in); got != want {
			t.Errorf("resolveRaidIcons(%q) = %q, want %q", in, got, want)
		}
	}
}
