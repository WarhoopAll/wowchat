package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	clog "github.com/charmbracelet/log"
)

var (
	mu sync.RWMutex

	mainLogger *clog.Logger
	chatLogger *clog.Logger
	chatFile   *os.File
	chatPrefix = clog.WithPrefix("CHAT")
)

func Init(debug bool, chatPath string) error {
	mu.Lock()
	defer mu.Unlock()

	level := clog.InfoLevel
	if debug {
		level = clog.DebugLevel
	}

	mainLogger = clog.NewWithOptions(os.Stderr, clog.Options{
		Level:        level,
		TimeFormat:   "15:04:05",
		CallerOffset: 0,
	})
	mainLogger.SetPrefix("APP")
	clog.SetDefault(mainLogger)
	clog.SetLevel(level)
	clog.SetTimeFormat("15:04:05")

	chatLogger = clog.NewWithOptions(io.Discard, clog.Options{
		Level:      clog.InfoLevel,
		TimeFormat: "2006-01-02 15:04:05",
	})
	chatLogger.SetPrefix("CHAT")
	chatPrefix = chatLogger

	chatFile = nil
	if chatPath != "" {
		if dir := filepath.Dir(chatPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create chat log dir %s: %w", dir, err)
			}
		}
		f, err := os.OpenFile(chatPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("open chat log %s: %w", chatPath, err)
		}
		chatFile = f
		chatLogger.SetOutput(io.MultiWriter(os.Stderr, chatFile))
	}
	return nil
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if chatFile != nil {
		err := chatFile.Close()
		chatFile = nil
		return err
	}
	return nil
}

func Chat() *clog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return chatPrefix
}

func Default() *clog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if mainLogger == nil {
		return clog.Default()
	}
	return mainLogger
}

func prefixed(name string) *clog.Logger {
	return Default().WithPrefix(name)
}

func WoW() *clog.Logger { return prefixed("WoW") }

func Realm() *clog.Logger { return prefixed("Realm") }

func Discord() *clog.Logger { return prefixed("Discord") }

func System() *clog.Logger { return prefixed("System") }
