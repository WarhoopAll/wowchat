package discord

import (
	"fmt"
	"regexp"
	"strings"
)

type resolver struct {
	linkSite string
	links    []linkRule
}

type linkRule struct {
	key string
	re  *regexp.Regexp
}

const defaultLinkSite = "https://db.warhoop.su"

var raidIcons = map[string]string{
	"{star}":        "⭐",
	"{звезда}":      "⭐",
	"{skull}":       "💀",
	"{череп}":       "💀",
	"{cross}":       "❌",
	"{крест}":       "❌",
	"{circle}":      "⚫",
	"{круг}":        "⚫",
	"{moon}":        "🌙",
	"{полумесяц}":   "🌙",
	"{diamond}":     "♦️",
	"{ромб}":        "♦️",
	"{square}":      "◼️",
	"{квадрат}":     "◼️",
	"{triangle}":    "🔺",
	"{треугольник}": "🔺",
}

var (
	colorPass1 = regexp.MustCompile(`\|c[0-9a-fA-F]{8}(.*?)\|r`)
	colorPass2 = regexp.MustCompile(`\|c[0-9a-fA-F]{8}`)
)

func newResolver(linkSite string) *resolver {
	if linkSite == "" {
		linkSite = defaultLinkSite
	}
	rules := []linkRule{
		{"item", regexp.MustCompile(`\|.+?\|Hitem:(\d+):.+?\|h\[(.+?)]\|h\|r`)},
		{"spell", regexp.MustCompile(`\|.+?\|(?:Hspell|Henchant|Htalent)?:(\d+).*?\|h\[(.+?)]\|h\|r`)},
		{"quest", regexp.MustCompile(`\|.+?\|Hquest:(\d+):.+?\|h\[(.+?)]\|h\|r`)},
		{"achievement", regexp.MustCompile(`\|.+?\|Hachievement:(\d+):.+?\|h\[(.+?)]\|h\|r`)},
		{"spell", regexp.MustCompile(`\|Htrade:(\d+):.+?\|h\[(.+?)]\|h`)},
	}
	return &resolver{linkSite: linkSite, links: rules}
}

func (r *resolver) resolveLinks(message string) string {
	for _, rule := range r.links {
		re := rule.re
		key := rule.key
		message = re.ReplaceAllStringFunc(message, func(match string) string {
			m := re.FindStringSubmatch(match)
			if len(m) < 3 {
				return match
			}
			id, name := m[1], m[2]
			return fmt.Sprintf("[[%s]](%s?%s=%s)", name, r.linkSite, key, id)
		})
	}
	return message
}

func (r *resolver) resolveRaidIcons(message string) string {
	for token, emoji := range raidIcons {
		message = strings.ReplaceAll(message, token, emoji)
	}
	return message
}

func stripColorCoding(message string) string {
	message = colorPass1.ReplaceAllString(message, "$1")
	message = colorPass2.ReplaceAllString(message, "")
	return message
}
