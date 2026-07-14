package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/WarhoopAll/wowchat/internal/app"
	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/version"
)

func main() {
	checkConfig := flag.Bool("check-config", false, "validate configuration and exit")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version.Version)
		return
	}

	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(2)
	}

	if err := logger.Init(cfg.Debug, "logs/chat.log"); err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(2)
	}
	defer logger.Close()

	if *checkConfig {
		fmt.Println("configuration is valid")
		fmt.Printf("  app version: %s\n", version.Version)
		fmt.Printf("  version:     %s (build %d)\n", cfg.WoW.Version, cfg.WoW.Build)
		fmt.Printf("  platform:    %s\n", cfg.WoW.Platform)
		fmt.Printf("  realm:       %s @ %s:%d\n", cfg.WoW.Realm, cfg.WoW.Host, cfg.WoW.Port)
		fmt.Printf("  account:     %s\n", cfg.WoW.Account)
		fmt.Printf("  character:   %s\n", cfg.WoW.Character)
		fmt.Printf("  channels:    %v\n", cfg.Channels)
		return
	}

	logger.PrintBanner(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a := app.New(cfg)
	if err := a.Run(ctx); err != nil {
		logger.System().Error("exited", "err", err)
		cancel()
		os.Exit(1)
	}
}
