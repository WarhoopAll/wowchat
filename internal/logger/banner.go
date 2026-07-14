package logger

import (
	"fmt"
	"strings"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/version"
)

const (
	cyanBold = "\033[1;36m"
	cyan     = "\033[36m"
	green    = "\033[32m"
	red      = "\033[31m"
	yellow   = "\033[33m"
	reset    = "\033[0m"
)

func mark(on bool, onText string) string {
	if on {
		return green + onText + reset
	}
	return red + "✗ no" + reset
}

func clampList(items []string, budget int) string {
	if len(items) == 0 {
		return ""
	}
	var out []string
	total := 0
	for _, it := range items {
		if total+len(it)+2 > budget && len(out) > 0 {
			out = append(out, yellow+"…"+reset)
			break
		}
		out = append(out, it)
		total += len(it) + 2
	}
	return strings.Join(out, ", ")
}

func borderTop() string { return "╔" + strings.Repeat("═", 40) + "╗" }
func borderBot() string { return "╚" + strings.Repeat("═", 40) + "╝" }
func borderMid() string { return "╠" + strings.Repeat("═", 40) + "╣" }

func titleRow(text string) string {
	return fmt.Sprintf("║ %s%-38s%s ║", cyanBold, text, reset)
}

func row(label, value string) string {
	return fmt.Sprintf("║ %s%-11s%s %s ║", cyan, label, reset, value)
}

func pad(value string, width int) string {
	display := visibleLen(value)
	if display >= width {
		return value
	}
	return value + strings.Repeat(" ", width-display)
}

func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

func PrintBanner(cfg *config.Config) {
	const innerW = 38

	discord := mark(cfg.Discord.Token != "", "✓ on")
	proxy := red + "✗ off" + reset
	if cfg.ProxyURL != nil {
		proxyURL := cfg.ProxyURL.Redacted()
		if len(proxyURL) > innerW-15 {
			proxyURL = proxyURL[:innerW-16] + "…"
		}
		proxy = yellow + "⚑ " + proxyURL + reset
	}
	debug := mark(cfg.Debug, "✓ on")

	realmLine := fmt.Sprintf("%s @ %s:%d", cfg.WoW.Realm, cfg.WoW.Host, cfg.WoW.Port)
	clientLine := fmt.Sprintf("%s (build %d)", cfg.WoW.Version, cfg.WoW.Build)
	chanLine := fmt.Sprintf("%d", len(cfg.Channels))
	if len(cfg.Channels) > 0 {
		chanLine = fmt.Sprintf("%d  %s", len(cfg.Channels), clampList(cfg.Channels, 20))
	}

	fmt.Println()
	fmt.Println(cyan + borderTop() + reset)
	fmt.Println(titleRow("WoWChat  " + version.Version))
	fmt.Println(cyan + borderMid() + reset)
	fmt.Println(row("Realm", pad(realmLine, innerW-12)))
	fmt.Println(row("Account", pad(cfg.WoW.Account, innerW-12)))
	fmt.Println(row("Character", pad(cfg.WoW.Character, innerW-12)))
	fmt.Println(row("Client", pad(clientLine, innerW-12)))
	fmt.Println(row("Discord", pad(discord, innerW-12)))
	fmt.Println(row("Channels", pad(chanLine, innerW-12)))
	fmt.Println(row("Proxy", pad(proxy, innerW-12)))
	fmt.Println(row("Debug", pad(debug, innerW-12)))
	fmt.Println(cyan + borderBot() + reset)
	fmt.Println()
}
