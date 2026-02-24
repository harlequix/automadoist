package main

import (
	"github.com/harlequix/godoist"
)

type ReviewsConfig struct {
	Prefixes        []string        `koanf:"prefixes"`
	Purge           bool            `koanf:"purge"`
	Label           string          `koanf:"label"`
	NextItemsConfig NextItemsConfig `koanf:"next_items"`
	Clean           bool            `koanf:"clean"`
}

func defaultReviewsConfig(cfg NextItemsConfig) ReviewsConfig {

	return ReviewsConfig{
		Prefixes:        []string{"*"},
		Purge:           false,
		Label:           "review",
		NextItemsConfig: cfg,
		Clean:           true,
	}
}

func prepare(cfg ReviewsConfig, nextItemsConfig NextItemsConfig) NextItemsConfig {
	out := nextItemsConfig
	for _, prefix := range cfg.Prefixes {
		var filtered []string
		for _, skip := range out.SkipPrefixes {
			if skip != prefix {
				filtered = append(filtered, skip)
			}
		}
		out.SkipPrefixes = filtered
	}
	var filtered []string
	for _, label := range out.IgnoreLabels {
		if label != cfg.Label {
			filtered = append(filtered, label)
		}
	}
	out.IgnoreLabels = filtered
	return out
}

func reviews(client *godoist.Todoist, cfg ReviewsConfig) {
	entry_search := client.Projects.GetByName(cfg.NextItemsConfig.EntryPoint)
	if len(entry_search) != 1 {
		logger.Error("Entry point not found")
		return
	}
	NextItemsConfig := prepare(cfg, cfg.NextItemsConfig)

	entry := entry_search[0]
	projects := collectProjects(*entry)
	logger.Info("Processing reviews", "config", NextItemsConfig)
	var next_items []*godoist.Task
	for _, project := range projects {
		tasks := getNextTasks(project, NextItemsConfig)
		next_items = append(next_items, tasks...)
	}
	needsReviewTasks := []*godoist.Task{}
	reviewTasks := []*godoist.Task{}
	for _, item := range next_items {
		logger.Debug("Processing item", "item", item)
		if hasPrefix(item.Content, cfg.Prefixes) {
			reviewTasks = append(reviewTasks, item)
			if !hasLabel([]string{cfg.Label}, item) {
				logger.Debug("Adding label", "label", cfg.Label, "item", item)
				needsReviewTasks = append(needsReviewTasks, item)
			}
		}
	}

	var comparing = []*godoist.Task{}
	if cfg.Purge {
		comparing = client.Tasks.All()
	} else if cfg.Clean {
		comparing = GetTasks(projects)
	}
	var toRemove []*godoist.Task
	for _, task := range comparing {
		if hasLabel([]string{cfg.Label}, task) && !isTaskInList(task, reviewTasks) {
			toRemove = append(toRemove, task)
		}
	}
	runParallel(toRemove, func(task *godoist.Task) {
		logger.Debug("Removing label", "label", cfg.Label, "task", task)
		if err := task.RemoveLabel(cfg.Label); err != nil {
			logger.Error("Failed to remove label", "label", cfg.Label, "task", task.Content, "error", err)
		}
	})
	runParallel(needsReviewTasks, func(task *godoist.Task) {
		task.AddLabel(cfg.Label)
	})

}
