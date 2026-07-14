package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
)

func (c *Client) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	c.mu.RLock()
	self := c.selfID
	c.mu.RUnlock()
	if m.Author == nil || m.Author.ID == self || m.Author.Bot {
		return
	}

	if m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply {
		return
	}
	ch, err := s.State.Channel(m.ChannelID)
	if err != nil || ch == nil || ch.Type != discordgo.ChannelTypeGuildText {
		return
	}

	name := sanitizeName(c.effectiveName(m))
	message := c.composeMessage(m)
	if message == "" {
		return
	}

	if c.shouldIgnore(message) {
		logger.Chat().Info("Discord->WoW ignored (prefix)", "msg", message)
		return
	}

	channelName := strings.ToLower(ch.Name)
	channelID := m.ChannelID

	enableCmdChans := c.cfg.Discord.EnableCommandsChans
	if len(enableCmdChans) > 0 && !containsFold(enableCmdChans, channelName) {
	}

	c.mu.RLock()
	routes := c.discordToWow[channelName]
	if len(routes) == 0 {
		routes = c.discordToWow[channelID]
	}
	routes = append([]wowRoute(nil), routes...)
	c.mu.RUnlock()
	if len(routes) == 0 {
		return
	}

	c.gameMu.RLock()
	game := c.game
	c.gameMu.RUnlock()

	type destKey struct {
		tp      byte
		channel string
	}
	seen := map[destKey]bool{}

	for _, route := range routes {
		direct := c.shouldSendDirectly(message)

		var finalMessages []string
		if direct {
			finalMessages = []string{message}
		} else {
			finalMessages = splitUpMessageToWow(route.format, name, message)
		}

		tp := route.tp
		var chanPtr *string
		if route.channel != "" {
			cn := route.channel
			chanPtr = &cn
		}
		if direct {
			tp = config.ChatTypeSay
			chanPtr = nil
		}

		dest := ""
		if chanPtr != nil {
			dest = *chanPtr
		} else {
			dest = wowTypeName(tp)
		}
		key := destKey{tp: tp, channel: dest}
		if seen[key] {
			continue
		}
		seen[key] = true

		for _, fm := range finalMessages {
			if c.shouldFilter(fm) {
				logger.Chat().Info("Discord->WoW FILTERED", "msg", fm)
				continue
			}
			logger.Chat().Info("Discord->WoW", "dest", dest, "msg", fm)
			if game == nil {
				logger.Chat().Warn("cannot send message; not connected to WoW")
				continue
			}
			if err := game.SendChatMessage(tp, fm, chanPtr); err != nil {
				logger.Discord().Error("discord->wow send failed", "err", err)
			}
		}
	}
}

func (c *Client) effectiveName(m *discordgo.MessageCreate) string {
	if m.Member != nil && m.Member.Nick != "" {
		return m.Member.Nick
	}
	if m.Author.GlobalName != "" {
		return m.Author.GlobalName
	}
	return m.Author.Username
}

var emojiRe = regexp.MustCompile(`<a?:\w+:\d+>|[\x{1F000}-\x{1FAFF}\x{2190}-\x{21FF}\x{2300}-\x{23FF}\x{24C2}\x{2500}-\x{25FF}\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{1F1E6}-\x{1F1FF}\x{200D}\x{20E3}\x{FE0F}]`)

func stripEmoji(s string) string {
	return emojiRe.ReplaceAllString(s, "")
}

func (c *Client) composeMessage(m *discordgo.MessageCreate) string {
	parts := make([]string, 0, 1+len(m.Attachments))
	content := strings.TrimSpace(m.ContentWithMentionsReplaced())
	if c.cfg.Discord.StripEmoji {
		content = stripEmoji(content)
	}
	if content != "" {
		parts = append(parts, content)
	}
	for _, a := range m.Attachments {
		if a.URL != "" {
			parts = append(parts, a.URL)
		}
	}
	return strings.Join(parts, " ")
}

func (c *Client) shouldIgnore(message string) bool {
	for _, p := range c.cfg.Discord.IgnorePrefixes {
		if strings.HasPrefix(message, p) {
			return true
		}
	}
	return false
}

func (c *Client) shouldSendDirectly(message string) bool {
	if !strings.HasPrefix(message, ".") || !c.cfg.Discord.EnableDotCommands {
		return false
	}
	trimmed := strings.ToLower(message[1:])
	wl := c.cfg.Discord.DotCommandsAllow
	if len(wl) == 0 {
		return true
	}
	for _, item := range wl {
		if strings.EqualFold(item, trimmed) {
			return true
		}
	}
	for _, item := range wl {
		if strings.HasSuffix(item, "*") {
			prefix := strings.ToLower(item[:len(item)-1])
			if strings.HasPrefix(trimmed, prefix) {
				return true
			}
		}
	}
	return false
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

func wowTypeName(tp byte) string {
	switch tp {
	case 0x00:
		return "System"
	case 0x01:
		return "Say"
	case 0x04:
		return "Guild"
	case 0x05:
		return "Officer"
	case 0x06:
		return "Yell"
	case 0x07:
		return "Whisper"
	case 0x0A:
		return "Emote"
	case 0x11:
		return "Channel"
	default:
		return "Unknown"
	}
}
