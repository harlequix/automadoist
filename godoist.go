package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/harlequix/godoist"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/cliflagv2"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v2"
)

type config struct {
	Token         string            `koanf:"token"`
	NextItems     NextItemsConfig   `koanf:"next_items"`
	ReviewsConfig ReviewsConfig     `koanf:"reviews"`
	DefaultTags   DefaultTagsConfig `koanf:"default_tags"`
}

func (c config) Verify() error {
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

var logger *slog.Logger
var level *slog.LevelVar
var k = koanf.New(".")
var ENV_PREFIX = "GODOIST_"

func init() {
	level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		AddSource: true,
	}))
	slog.SetDefault(logger)
}

var defaultConfig = config{
	Token:         "",
	NextItems:     defaultNextItemsConfig(),
	ReviewsConfig: defaultReviewsConfig(NextItemsConfig{}),
}

func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}

func getConfig(c *cli.Context) (*config, error) {
	var cfg config = defaultConfig
	var loglevel string
	if c.Bool("debug") {
		loglevel = "debug"
	} else {
		loglevel = c.String("log-level")
	}
	lvl, err := ParseLevel(loglevel)
	if err != nil {
		return nil, err
	}
	level.Set(lvl)
	logger.Info("Todoist client created")
	if c.String("config") != "" {
		logger.Debug("Loading configuration from file", "file", c.String("config"))
		if err := k.Load(file.Provider(c.String("config")), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}
	p := cliflagv2.Provider(c, "godoist")
	if err := k.Load(p, nil); err != nil {
		return nil, fmt.Errorf("loading cli flags: %w", err)
	}
	if err := k.Load(env.Provider(ENV_PREFIX, ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, ENV_PREFIX)), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}
	logger.Debug("Loaded configuration", "config", k.Raw())
	err = k.Unmarshal("", &cfg)
	if err != nil {
		return nil, err
	}
	err = cfg.Verify()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	start := time.Now()

	app := &cli.App{
		Name:  "godoist",
		Usage: "Manage Todoist tasks",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
				Value:   false,
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to the configuration file",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "Todoist API token",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Log level",
				Value:   "warn",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "next_items",
				Usage: "Manage next items in Todoist",
				Action: func(c *cli.Context) error {
					cfg, err := getConfig(c)
					if err != nil {
						return err
					}
					logger.Debug("loaded and verified config", "config", cfg)
					client := godoist.NewTodoist(cfg.Token)
					if err := client.Sync(); err != nil {
						return err
					}
					process_next_items(client, cfg.NextItems)
					client.API.Commit()
					finish := time.Now()
					logger.Info("Finished", "duration", finish.Sub(start))
					return nil
				},
			},
			{
				Name:  "default_tags",
				Usage: "Configure default tags for projects",
				Action: func(c *cli.Context) error {
					cfg, err := getConfig(c)
					if err != nil {
						return err
					}
					client := godoist.NewTodoist(cfg.Token)
					if err := client.Sync(); err != nil {
						return err
					}
					return defaultTagsCommand(client, cfg.DefaultTags)
				},
			},
			{
				Name:  "reviews",
				Usage: "Manage review items in Todoist",
				Action: func(c *cli.Context) error {
					cfg, err := getConfig(c)
					if err != nil {
						return err
					}
					logger.Debug("loaded and verified config", "config", cfg)
					client := godoist.NewTodoist(cfg.Token)
					if err := client.Sync(); err != nil {
						return err
					}
					if cfg.ReviewsConfig.NextItemsConfig.EntryPoint == "" {
						cfg.ReviewsConfig.NextItemsConfig = cfg.NextItems
					}
					reviews(client, cfg.ReviewsConfig)
					client.API.Commit()
					finish := time.Now()
					logger.Info("Finished", "duration", finish.Sub(start))
					return nil
				},
			},
		},
	}

	logger.Debug("Starting godoist")
	err := app.Run(os.Args)
	if err != nil {
		logger.Error("Error", "error", err)

	}
}
