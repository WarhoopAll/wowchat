package discord

import (
	"strings"
	"time"
)

func getTime() string { return time.Now().Format("15:04:05") }

func escapeDiscordMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"~", "\\~",
	)
	return replacer.Replace(s)
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(name, "|", "||")
}

func applyFormat(format, timeStr, user, message, target string) string {
	replacer := strings.NewReplacer(
		"%time", timeStr,
		"%user", user,
		"%message", message,
		"%target", target,
	)
	return replacer.Replace(format)
}

func splitByLength(message string, maxLength int) []string {
	if maxLength <= 0 {
		return []string{message}
	}
	var out []string
	tmp := message
	for len([]rune(tmp)) > maxLength {
		runes := []rune(tmp)
		sub := string(runes[:maxLength])
		spaceIndex := strings.LastIndex(sub, " ")
		if spaceIndex == -1 {
			out = append(out, sub)
			tmp = string(runes[maxLength:])
		} else {
			out = append(out, sub[:spaceIndex])
			tmp = string([]rune(tmp)[spaceIndex+1:])
		}
	}
	if tmp != "" {
		out = append(out, tmp)
	}
	return out
}

func splitUpMessageToWow(format, name, message string) []string {
	ts := getTime()
	base := applyFormat(format, ts, name, "", "")
	maxTmpLen := 255 - len([]rune(base))
	if maxTmpLen < 1 {
		maxTmpLen = 1
	}
	parts := splitByLength(message, maxTmpLen)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		formatted := applyFormat(format, ts, name, p, "")
		if strings.HasPrefix(formatted, ".") {
			formatted = " " + formatted
		}
		out = append(out, formatted)
	}
	return out
}
