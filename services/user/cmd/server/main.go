package main

import (
	"context"
	"flag"
	"log/slog"

	"github.com/yaninyzwitty/chat/pkg/config"
)

func main() {
	cp := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.Config{}

	if *cp != "" {
		cfg.LoadConfig(*cp)
	} else {
		// fallback if no config path is given
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

}
