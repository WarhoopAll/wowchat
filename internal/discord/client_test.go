package discord

import (
	"regexp"
	"testing"

	"github.com/WarhoopAll/wowchat/internal/config"
)

func TestShouldSendDirectly(t *testing.T) {
	c := &Client{cfg: &config.Config{Discord: config.Discord{
		EnableDotCommands: true,
		DotCommandsAllow:  nil,
	}}}
	if !c.shouldSendDirectly(".who") {
		t.Errorf("empty whitelist should allow any dot command")
	}
	if c.shouldSendDirectly("hello") {
		t.Errorf("non-dot message should not be direct")
	}

	c.cfg.Discord.DotCommandsAllow = []string{"who", "lookup*"}
	if !c.shouldSendDirectly(".who") {
		t.Errorf("whitelisted exact command should be direct")
	}
	if !c.shouldSendDirectly(".lookup thunderfury") {
		t.Errorf("wildcard prefix should match")
	}
	if c.shouldSendDirectly(".server info") {
		t.Errorf("non-whitelisted command should not be direct")
	}

	c.cfg.Discord.EnableDotCommands = false
	if c.shouldSendDirectly(".who") {
		t.Errorf("dot commands disabled should never be direct")
	}
}

func TestShouldFilter(t *testing.T) {
	c := &Client{
		cfg:     &config.Config{FiltersEnabled: true},
		filters: []*regexp.Regexp{regexp.MustCompile("^(?:.*spam.*)$")},
	}
	if !c.shouldFilter("this is spam here") {
		t.Errorf("spam should be filtered")
	}
	if c.shouldFilter("clean message") {
		t.Errorf("clean message should not be filtered")
	}

	c.cfg.FiltersEnabled = false
	if c.shouldFilter("this is spam here") {
		t.Errorf("filters disabled should never filter")
	}
}

func TestWantsWoWAndSendFromWoWRouting(t *testing.T) {
	lfg := "lookingforgroup"
	c := &Client{
		cfg:  &config.Config{},
		res:  newResolver(""),
		wowToDiscord: map[wowKey][]discordTarget{
			{tp: 0x11, channel: "lookingforgroup"}: {{channelID: "123", format: "[%user]: %message"}},
			{tp: 0x00, channel: ""}:                {{channelID: "123", format: "[SYSTEM]: %message"}},
		},
	}
	if !c.WantsWoW(0x11, &lfg) {
		t.Errorf("channel route should be wanted")
	}
	if !c.WantsWoW(0x00, nil) {
		t.Errorf("system route should be wanted")
	}
	other := "trade"
	if c.WantsWoW(0x11, &other) {
		t.Errorf("unconfigured channel should not be wanted")
	}
}
