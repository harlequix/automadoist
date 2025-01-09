package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/harlequix/godoist"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v2"
)

type config struct {
	Token     string
	NextItems NextItemsConfig
}

var logger *slog.Logger
var level *slog.LevelVar
var k = koanf.New(".")

func init() {
	level = new(slog.LevelVar)
	level.Set(slog.LevelDebug)
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

var defaultConfig = config{
	Token:     "86c7b47b2213eb37a694a19d095dfc46892f4614",
	NextItems: defaultNextItemsConfig(),
}

func main() {
	logger.Debug("Starting godoist")
	start := time.Now()
	var cfg config = defaultConfig
	k.Load(file.Provider("config.yaml"), yaml.Parser())
	k.Print()
	k.Unmarshal("", &cfg)
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "next_items",
				Usage: "Manage next items in Todoist",
				Action: func(c *cli.Context) error {
					client := godoist.NewTodoist(cfg.Token)
					logger.Info("Todoist client created")
					client.Sync()
					next_items(client, cfg.NextItems)
					finish := time.Now()
					logger.Info("Finished", "duration", finish.Sub(start))
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Error("Error", "error", err)

	}
}
