package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WarhoopAll/wowchat/internal/proxy"
)

type ProxyConfig struct {
	URL                string
	DiscordConnect     bool
	RealmConnect       bool
	InsecureSkipVerify bool
}

type Platform string

const (
	PlatformMac     Platform = "Mac"
	PlatformWindows Platform = "Windows"
)

type Config struct {
	Discord        Discord
	WoW            WoW
	Proxy          ProxyConfig
	ProxyURL       *url.URL
	Routes         []Route
	Channels       []string
	FiltersEnabled bool
	FilterPatterns []string
	Debug          bool
}

type Route struct {
	Direction   string
	WowType     byte
	WowChannel  string
	WowFormat   string
	DiscordChan string
	DiscordFmt  string
}

type Discord struct {
	Token               string
	EnableDotCommands   bool
	DotCommandsAllow    []string
	EnableCommandsChans []string
	TagFailedNotify     bool
	ItemDatabase        string
	IgnorePrefixes      []string
	StripEmoji          bool
}

type WoW struct {
	Platform  Platform
	Version   string
	Build     int
	Host      string
	Port      int
	Realm     string
	Account   string
	Password  string
	Character string
}

var (
	varRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	errMissing = errors.New("missing required value")
)

func Load(path string) (*Config, error) {
	values, err := readDotEnv(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			values[kv[:i]] = kv[i+1:]
		}
	}
	return build(values)
}

func readDotEnv(path string) (map[string]string, error) {
	values := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return values, nil
		}
		return nil, err
	}
	defer f.Close()

	resolve := func(raw string) string {
		return varRef.ReplaceAllStringFunc(raw, func(m string) string {
			name := m[2 : len(m)-1]
			if v, ok := values[name]; ok {
				return v
			}
			return os.Getenv(name)
		})
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if n := len(val); n >= 2 {
			first, last := val[0], val[n-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : n-1]
			}
		}
		values[key] = resolve(val)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func build(v map[string]string) (*Config, error) {
	c := &Config{}
	var problems []string
	require := func(key string) string {
		s := strings.TrimSpace(v[key])
		if s == "" {
			problems = append(problems, key)
		}
		return s
	}

	account := require("WOW_ACCOUNT")
	password := require("WOW_PASSWORD")
	character := require("WOW_CHARACTER")
	host := require("WOW_REALMLIST_HOST")
	realm := require("WOW_REALM")
	version := require("WOW_VERSION")

	portStr := strings.TrimSpace(v["WOW_REALMLIST_PORT"])
	if portStr == "" {
		portStr = "3724"
	}
	port, perr := strconv.Atoi(portStr)
	if perr != nil || port < 1 || port > 65535 {
		problems = append(problems, "WOW_REALMLIST_PORT")
	}

	platform := PlatformMac
	switch strings.ToLower(strings.TrimSpace(v["WOW_PLATFORM"])) {
	case "win", "windows":
		platform = PlatformWindows
	}

	build, berr := buildForVersion(version)
	if berr != nil {
		problems = append(problems, "WOW_VERSION")
	}

	c.WoW = WoW{
		Platform:  platform,
		Version:   version,
		Build:     build,
		Host:      host,
		Port:      port,
		Realm:     realm,
		Account:   convertToUpper(account),
		Password:  password,
		Character: character,
	}

	c.Discord = Discord{
		Token:               strings.TrimSpace(v["DISCORD_TOKEN"]),
		EnableDotCommands:   parseBool(v["DISCORD_ENABLE_DOT_COMMANDS"], true),
		EnableCommandsChans: splitSpaces(v["DISCORD_ENABLE_COMMANDS_CHANNELS"]),
		DotCommandsAllow:    splitList(v["DISCORD_DOT_COMMANDS_WHITELIST"], ','),
		TagFailedNotify:     parseBool(v["DISCORD_ENABLE_TAG_FAILED_NOTIFICATIONS"], true),
		ItemDatabase:        itemDatabase(v),
		IgnorePrefixes:      ignorePrefixes(v),
		StripEmoji:          parseBool(v["DISCORD_STRIP_EMOJI"], true),
	}

	c.FiltersEnabled = parseBool(v["FILTERS_ENABLED"], false)
	c.FilterPatterns = splitSpaces(v["FILTER_PATTERNS"])
	c.Debug = parseBool(v["DEBUG"], false)

	proxyURL, perr := parseProxyURL(strings.TrimSpace(v["PROXY_URL"]))
	if perr != nil {
		problems = append(problems, "PROXY_URL")
	}
	c.ProxyURL = proxyURL
	c.Proxy = ProxyConfig{
		URL:                strings.TrimSpace(v["PROXY_URL"]),
		DiscordConnect:     parseBool(v["PROXY_DISCORD_CONNECT"], false),
		RealmConnect:       parseBool(v["PROXY_REALM_CONNECT"], false),
		InsecureSkipVerify: parseBool(v["PROXY_INSECURE_SKIP_VERIFY"], false),
	}

	c.Routes, c.Channels = collectRoutes(v)

	if len(problems) > 0 {
		return nil, fmt.Errorf("%w: %s", errMissing, strings.Join(problems, ", "))
	}
	return c, nil
}

func collectRoutes(v map[string]string) (routes []Route, channels []string) {
	count, err := strconv.Atoi(strings.TrimSpace(v["CHAT_ROUTE_COUNT"]))
	if err != nil || count <= 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	for i := 1; i <= count; i++ {
		pfx := fmt.Sprintf("CHAT_ROUTE_%d_", i)
		dir := strings.TrimSpace(v[pfx+"DIRECTION"])
		wowTypeStr := strings.TrimSpace(v[pfx+"WOW_TYPE"])
		r := Route{
			Direction:   dir,
			WowType:     ParseChatType(wowTypeStr),
			WowChannel:  strings.TrimSpace(v[pfx+"WOW_CHANNEL"]),
			WowFormat:   strings.TrimSpace(v[pfx+"WOW_FORMAT"]),
			DiscordChan: strings.TrimSpace(v[pfx+"DISCORD_CHANNEL"]),
			DiscordFmt:  strings.TrimSpace(v[pfx+"DISCORD_FORMAT"]),
		}
		routes = append(routes, r)
		if r.WowType == ChatTypeChannel && r.WowChannel != "" && !seen[r.WowChannel] {
			seen[r.WowChannel] = true
			channels = append(channels, r.WowChannel)
		}
	}
	return routes, channels
}

const (
	ChatTypeSystem  byte = 0x00
	ChatTypeSay     byte = 0x01
	ChatTypeGuild   byte = 0x04
	ChatTypeOfficer byte = 0x05
	ChatTypeYell    byte = 0x06
	ChatTypeWhisper byte = 0x07
	ChatTypeEmote   byte = 0x0A
	ChatTypeChannel byte = 0x11
)

func ParseChatType(s string) byte {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "system":
		return ChatTypeSystem
	case "say":
		return ChatTypeSay
	case "guild":
		return ChatTypeGuild
	case "officer":
		return ChatTypeOfficer
	case "yell":
		return ChatTypeYell
	case "emote":
		return ChatTypeEmote
	case "whisper":
		return ChatTypeWhisper
	case "channel", "custom":
		return ChatTypeChannel
	default:
		return 0xFF
	}
}

func buildForVersion(version string) (int, error) {
	table := map[string]int{
		"1.6.1": 4544, "1.6.2": 4565, "1.6.3": 4620, "1.7.1": 4695,
		"1.8.4": 4878, "1.9.4": 5086, "1.10.2": 5302, "1.11.2": 5464,
		"1.12.1": 5875, "1.12.2": 6005, "1.12.3": 6141,
		"2.4.3": 8606,
		"3.2.2": 10505, "3.3.0": 11159, "3.3.2": 11403, "3.3.3": 11723, "3.3.5": 12340,
		"4.3.4": 15595,
		"5.4.8": 18414,
	}
	if b, ok := table[version]; ok {
		return b, nil
	}
	return 0, fmt.Errorf("unsupported wow version %q", version)
}

func convertToUpper(account string) string {
	b := []byte(account)
	for i, ch := range b {
		if ch >= 'a' && ch <= 'z' {
			b[i] = ch - 'a' + 'A'
		}
	}
	return string(b)
}

func ignorePrefixes(v map[string]string) []string {
	p := splitList(v["DISCORD_IGNORE_PREFIXES"], ',')
	if len(p) == 0 {
		return []string{"!"}
	}
	return p
}

func itemDatabase(v map[string]string) string {
	if s := strings.TrimSpace(v["ITEM_DATABASE"]); s != "" {
		return s
	}
	return strings.TrimSpace(v["DISCORD_ITEM_DATABASE"])
}

func parseProxyURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid PROXY_URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q (want http, https or socks5)", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("PROXY_URL missing host")
	}
	return u, nil
}

func (c *Config) DialTCP(target string, viaProxy bool) (net.Conn, error) {
	if viaProxy && c.ProxyURL != nil {
		return proxy.NewDialer(c.ProxyURL, c.Proxy.InsecureSkipVerify).Dial(target)
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	return d.Dial("tcp", target)
}

func (c *Config) ProxyDialer() *proxy.Dialer {
	if !c.Proxy.DiscordConnect || c.ProxyURL == nil {
		return nil
	}
	return proxy.NewDialer(c.ProxyURL, c.Proxy.InsecureSkipVerify)
}

func parseBool(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "y", "yes":
		return true
	case "false", "0", "n", "no":
		return false
	default:
		return def
	}
}

func splitSpaces(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p)
	}
	return out
}

func splitList(s string, sep rune) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == sep })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
