package discord

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
)

type GameSender interface {
	SendChatMessage(tp byte, message string, channel *string) error
}

type wowKey struct {
	tp      byte
	channel string
}

type discordTarget struct {
	channelID string
	format    string
}

type wowRoute struct {
	tp      byte
	channel string
	format  string
}

type Client struct {
	cfg          *config.Config
	session      *discordgo.Session
	res          *resolver
	filters      []*regexp.Regexp
	mu           sync.RWMutex
	wowToDiscord map[wowKey][]discordTarget
	discordToWow map[string][]wowRoute
	selfID       string
	gameMu       sync.RWMutex
	game         GameSender
}

func New(cfg *config.Config) (*Client, error) {
	if strings.TrimSpace(cfg.Discord.Token) == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is empty")
	}
	c := &Client{
		cfg:          cfg,
		res:          newResolver(cfg.Discord.ItemDatabase),
		wowToDiscord: map[wowKey][]discordTarget{},
		discordToWow: map[string][]wowRoute{},
	}
	if cfg.FiltersEnabled {
		for _, p := range cfg.FilterPatterns {
			re, err := regexp.Compile("^(?:" + p + ")$")
			if err != nil {
				logger.Discord().Warn("ignoring invalid filter pattern", "pattern", p, "err", err)
				continue
			}
			c.filters = append(c.filters, re)
		}
	}
	return c, nil
}

func (c *Client) Start() error {
	s, err := discordgo.New("Bot " + c.cfg.Discord.Token)
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}
	s.Identify.Intents = discordgo.MakeIntent(
		discordgo.IntentsGuilds |
			discordgo.IntentsGuildMessages |
			discordgo.IntentsGuildMembers |
			discordgo.IntentsMessageContent,
	)
	c.session = s

	if d := c.cfg.ProxyDialer(); d != nil {
		s.Client.Transport = d.HTTPTransport()
		s.Dialer = d.WSDialer()
		logger.Discord().Info("routing via proxy", "url", c.cfg.ProxyURL.Redacted())
	}

	s.AddHandler(c.onReady)
	s.AddHandler(c.onGuildCreate)
	s.AddHandler(c.onMessageCreate)

	if err := s.Open(); err != nil {
		return fmt.Errorf("open discord gateway: %w", err)
	}
	logger.Discord().Info("gateway connected")
	return nil
}

func (c *Client) Close() error {
	if c.session == nil {
		return nil
	}
	return c.session.Close()
}

func (c *Client) SetGame(g GameSender) {
	c.gameMu.Lock()
	c.game = g
	c.gameMu.Unlock()
}

func (c *Client) onReady(s *discordgo.Session, r *discordgo.Ready) {
	c.mu.Lock()
	c.selfID = r.User.ID
	c.mu.Unlock()
	logger.Discord().Info("logged in", "user", r.User.Username+"#"+r.User.Discriminator)
	c.buildRoutes()
}

func (c *Client) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	c.buildRoutes()
}

func (c *Client) buildRoutes() {
	type chEntry struct {
		id   string
		name string // lowercased
	}
	var channels []chEntry
	for _, g := range c.session.State.Guilds {
		for _, ch := range g.Channels {
			if ch.Type == discordgo.ChannelTypeGuildText {
				channels = append(channels, chEntry{id: ch.ID, name: strings.ToLower(ch.Name)})
			}
		}
	}

	w2d := map[wowKey][]discordTarget{}
	d2w := map[string][]wowRoute{}

	for _, r := range c.cfg.Routes {
		want := strings.ToLower(r.DiscordChan)
		for _, ch := range channels {
			if ch.name != want && ch.id != r.DiscordChan {
				continue
			}
			if r.Direction == "both" || r.Direction == "wow_to_discord" {
				key := wowKey{tp: r.WowType, channel: strings.ToLower(r.WowChannel)}
				w2d[key] = append(w2d[key], discordTarget{channelID: ch.id, format: r.DiscordFmt})
			}
			if r.Direction == "both" || r.Direction == "discord_to_wow" {
				route := wowRoute{tp: r.WowType, channel: r.WowChannel, format: r.WowFormat}
				d2w[ch.name] = append(d2w[ch.name], route)
				d2w[ch.id] = append(d2w[ch.id], route)
			}
		}
	}

	c.mu.Lock()
	c.wowToDiscord = w2d
	c.discordToWow = d2w
	c.mu.Unlock()

	if len(w2d) == 0 && len(d2w) == 0 {
		logger.Discord().Warn("no configured channels matched any Discord text channel")
	} else {
		logger.Discord().Info("routes ready", "wowToDiscord", len(w2d), "discordToWow", len(d2w))
	}
}

func (c *Client) SendFromWoW(from *string, message string, wowType byte, wowChannel *string) {
	key := wowKey{tp: wowType}
	if wowChannel != nil {
		key.channel = strings.ToLower(*wowChannel)
	}
	c.mu.RLock()
	targets := append([]discordTarget(nil), c.wowToDiscord[key]...)
	c.mu.RUnlock()
	if len(targets) == 0 {
		return
	}

	parsed := stripColorCoding(c.res.resolveLinks(c.res.resolveRaidIcons(message)))

	user := ""
	if from != nil {
		user = *from
	}
	target := ""
	if wowChannel != nil {
		target = *wowChannel
	}

	for _, t := range targets {
		body := parsed
		if from != nil {
			body = escapeDiscordMarkdown(body)
		}
		formatted := applyFormat(t.format, getTime(), user, body, target)
		if c.shouldFilter(formatted) {
			logger.Chat().Info("WoW->Discord FILTERED", "msg", formatted)
			continue
		}
		logger.Chat().Info("WoW->Discord", "msg", formatted)
		c.sendMessage(t.channelID, formatted)
	}
}

func (c *Client) WantsWoW(wowType byte, wowChannel *string) bool {
	key := wowKey{tp: wowType}
	if wowChannel != nil {
		key.channel = strings.ToLower(*wowChannel)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.wowToDiscord[key]) > 0
}

func (c *Client) sendMessage(channelID, content string) {
	for _, part := range splitByLength(content, 2000) {
		if _, err := c.session.ChannelMessageSend(channelID, part); err != nil {
			logger.Discord().Error("send failed", "channel", channelID, "err", err)
		}
	}
}

func (c *Client) shouldFilter(message string) bool {
	if !c.cfg.FiltersEnabled {
		return false
	}
	for _, re := range c.filters {
		if re.MatchString(message) {
			return true
		}
	}
	return false
}
