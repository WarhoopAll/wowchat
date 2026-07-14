package app

import (
	"context"
	"time"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/discord"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/game"
	"github.com/WarhoopAll/wowchat/internal/wow/realm"
)

type App struct {
	cfg     *config.Config
	discord *discord.Client
}

func New(cfg *config.Config) *App { return &App{cfg: cfg} }

func (a *App) Run(ctx context.Context) error {
	const reconnectDelay = 10 * time.Second

	if a.cfg.Discord.Token != "" {
		d, err := discord.New(a.cfg)
		if err != nil {
			return err
		}
		if err := d.Start(); err != nil {
			return err
		}
		defer d.Close()
		a.discord = d
	} else {
		logger.System().Info("DISCORD_TOKEN not set; running without Discord relay")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := a.connectOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			logger.System().Warn("session ended", "err", err)
		}
		logger.System().Info("reconnecting", "delay", reconnectDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectDelay):
		}
	}
}

func (a *App) connectOnce(ctx context.Context) error {
	res, err := realm.New(a.cfg, "enUS").Login()
	if err != nil {
		return err
	}
	sess := game.NewSession(a.cfg, res.RealmID, res.RealmName, res.SessionKey)
	if a.discord != nil {
		sess.SetRelay(a.discord)
		a.discord.SetGame(sess)
		defer a.discord.SetGame(nil)
	}
	return sess.Run(ctx, res.Host, res.Port)
}
