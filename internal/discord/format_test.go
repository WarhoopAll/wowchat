package discord

import (
	"strings"
	"testing"
)

func TestApplyFormat(t *testing.T) {
	got := applyFormat("[%target] [%user]: %message", "12:00:00", "Bob", "hello", "LookingForGroup")
	want := "[LookingForGroup] [Bob]: hello"
	if got != want {
		t.Errorf("applyFormat = %q, want %q", got, want)
	}
}

func TestEscapeDiscordMarkdown(t *testing.T) {
	got := escapeDiscordMarkdown("a`b*c_d~e")
	want := "a\\`b\\*c\\_d\\~e"
	if got != want {
		t.Errorf("escape = %q, want %q", got, want)
	}
}

func TestSanitizeName(t *testing.T) {
	if got := sanitizeName("a|b"); got != "a||b" {
		t.Errorf("sanitizeName = %q", got)
	}
}

func TestSplitByLength(t *testing.T) {
	parts := splitByLength("aaaa bbbb cccc", 9)
	if len(parts) != 2 || parts[0] != "aaaa" || parts[1] != "bbbb cccc" {
		t.Errorf("splitByLength = %#v", parts)
	}

	parts = splitByLength("aaaaaaaaaa", 4)
	if len(parts) != 3 || parts[0] != "aaaa" || parts[2] != "aa" {
		t.Errorf("hard split = %#v", parts)
	}
}

func TestSplitUpMessageToWowDotEscape(t *testing.T) {
	out := splitUpMessageToWow("%message", "Bob", ".server info")
	if len(out) != 1 || !strings.HasPrefix(out[0], " .server") {
		t.Errorf("dot escape failed: %#v", out)
	}
}
