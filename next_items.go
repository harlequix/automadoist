package main

import (
	"sort"
	"strings"
	"time"

	"github.com/harlequix/godoist"
)

type NextItemsConfig struct {
	EntryPoint       string         `koanf:"entry_point"`
	SkipPrefixes     []string       `koanf:"skip_prefixes"`
	Recursive        bool           `koanf:"recursive"`
	SequentialMarker string         `koanf:"sequential_marker"`
	SkipDeadline     string         `koanf:"skip_deadline"`
	ManagedLabels    []string       `koanf:"managed_labels"`
	IgnoreLabels     []string       `koanf:"ignore_labels"`
	Prune            bool           `koanf:"prune"`
	ColorPriority    map[string]int `koanf:"color_priority"`
	ContextLabels    []string       `koanf:"context_labels"`
}

func defaultNextItemsConfig() NextItemsConfig {
	return NextItemsConfig{
		EntryPoint:       "projects",
		SkipPrefixes:     []string{"*"},
		Recursive:        true,
		SequentialMarker: "!",
		SkipDeadline:     "not_overdue",
		ManagedLabels:    []string{"next"},
		IgnoreLabels:     []string{"waiting", "review"},
		Prune:            true,
	}
}

func process_next_items(client *godoist.Todoist, cfg NextItemsConfig) {
	logger.Debug("Processing next items", "config", cfg)
	logger.Debug("Entry point", "entry_point", cfg.EntryPoint)
	entry_search := client.Projects.GetByName(cfg.EntryPoint)
	if len(entry_search) != 1 {
		logger.Error("Entry point not found")
		return
	}
	entry := entry_search[0]

	allSubProjects := collectProjects(*entry)
	allTasks := client.Tasks.All()
	nextTasks := []*godoist.Task{}
	for _, project := range allSubProjects {
		tasks := getNextTasks(project, cfg)
		nextTasks = append(nextTasks, tasks...)
	}
	var hasManagedLabel []*godoist.Task
	for _, task := range allTasks {
		if hasLabel(cfg.ManagedLabels, task) {
			hasManagedLabel = append(hasManagedLabel, task)
		}
	}

	var needRemoval []*godoist.Task
	for _, task := range hasManagedLabel {
		if !isTaskInList(task, nextTasks) {
			needRemoval = append(needRemoval, task)
		}
	}

	var needAddition []*godoist.Task
	for _, t := range nextTasks {
		if !hasLabel(cfg.ManagedLabels, t) {
			needAddition = append(needAddition, t)
		}
	}

	// Precompute project lookup maps for context operations
	projectTags, projectColors := buildProjectMaps(allSubProjects)
	contextEnabled := len(cfg.ContextLabels) > 0

	// Phase 1: Tasks LOSING @next
	runParallel(needRemoval, func(t *godoist.Task) {
		logger.Debug("Processing removal", "task", t.Content, "label", cfg.ManagedLabels[0])

		// Save context if task has customizations
		if contextEnabled {
			defaultLabels, defaultPriority := computeExpectedDefaults(t, projectTags, projectColors, cfg.ColorPriority, cfg.ContextLabels)
			if hasCustomizations(t, defaultLabels, defaultPriority, cfg.ContextLabels) {
				logger.Debug("Saving context for task", "task", t.Content)
				if err := saveContext(t, cfg.ContextLabels); err != nil {
					logger.Error("Failed to save context", "task", t.Content, "error", err)
				}
			}
		}

		// Strip labels: keep only ignore labels
		retained := computeRetainedLabels(t, cfg.IgnoreLabels)
		if err := t.Update("labels", retained); err != nil {
			logger.Error("Failed to update labels", "task", t.Content, "error", err)
		}

		// Reset priority
		if t.Priority != godoist.VERY_LOW {
			if err := t.Update("priority", godoist.VERY_LOW); err != nil {
				logger.Error("Failed to update priority", "task", t.Content, "error", err)
			}
		}
	})

	// Phase 2: Tasks GAINING @next
	runParallel(needAddition, func(t *godoist.Task) {
		logger.Debug("Processing addition", "task", t.Content, "label", cfg.ManagedLabels[0])

		// Check for saved context
		var saved *taskContext
		if contextEnabled {
			var err error
			saved, err = restoreContext(t)
			if err != nil {
				logger.Error("Failed to restore context", "task", t.Content, "error", err)
			}
		}

		if saved != nil {
			// Restore from saved context
			labelSet := toSet(t.Labels)
			labelSet[cfg.ManagedLabels[0]] = true
			for _, label := range saved.Labels {
				labelSet[label] = true
			}
			var labels []string
			for label := range labelSet {
				labels = append(labels, label)
			}
			if err := t.Update("labels", labels); err != nil {
				logger.Error("Failed to update labels", "task", t.Content, "error", err)
			}
			if err := t.Update("priority", saved.Priority); err != nil {
				logger.Error("Failed to update priority", "task", t.Content, "error", err)
			}
			if err := t.DeleteContext(); err != nil {
				logger.Error("Failed to delete context", "task", t.Content, "error", err)
			}
			logger.Debug("Restored context for task", "task", t.Content, "labels", saved.Labels, "priority", saved.Priority)
		} else {
			// Apply defaults for newly qualifying tasks
			labelSet := toSet(t.Labels)
			labelSet[cfg.ManagedLabels[0]] = true
			if tags, ok := projectTags[t.ProjectID]; ok {
				for _, tag := range tags {
					labelSet[tag] = true
				}
			}
			var labels []string
			for label := range labelSet {
				labels = append(labels, label)
			}
			if err := t.Update("labels", labels); err != nil {
				logger.Error("Failed to update labels", "task", t.Content, "error", err)
			}

			// Apply color priority only if currently VERY_LOW
			if t.Priority == godoist.VERY_LOW {
				if color, ok := projectColors[t.ProjectID]; ok {
					if priority, ok := cfg.ColorPriority[color]; ok {
						logger.Debug("Setting priority from project color", "task", t.Content, "color", color, "priority", priority)
						if err := t.Update("priority", godoist.PRIORITY_LEVEL(priority)); err != nil {
							logger.Error("Failed to update priority", "task", t.Content, "error", err)
						}
					}
				}
			}
		}
	})
}

func collectProjects(project godoist.Project) []godoist.Project {
	var allProjects []godoist.Project
	allProjects = append(allProjects, project)
	for _, subproject := range project.GetChildren() {
		allProjects = append(allProjects, collectProjects(*subproject)...)
	}
	return allProjects
}

func isTaskInList(task *godoist.Task, taskList []*godoist.Task) bool {
	for _, i := range taskList {
		if task.ID == i.ID {
			return true
		}
	}
	return false
}

func GetTasks(projects []godoist.Project) []*godoist.Task {
	var tasks []*godoist.Task
	for _, project := range projects {
		tasks = append(tasks, project.GetTasks()...)
	}
	return tasks
}

func hasPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func hasLabel(labels []string, task *godoist.Task) bool {
	for _, taskLabel := range task.Labels {
		for _, label := range labels {
			if taskLabel == label {
				return true
			}
		}
	}
	return false
}
func getNextTasks(project godoist.Project, cfg NextItemsConfig) []*godoist.Task {
	tasks := project.GetTasks()
	now := time.Now()
	var nextTasks []*godoist.Task
	var working_on []*godoist.Task
	logger.Debug("Number of task in project", "project", project.Name, "#", len(tasks), "color", project.Color)
	for _, task := range tasks {
		if task.ParentID == "" {
			working_on = append(working_on, task)
		}
	}
	for len(working_on) > 0 {
		task := working_on[0]
		working_on = working_on[1:]
		name := task.Content
		subtasks := task.GetChildren()
		//TODO: implement switch for sequential order
		sort.Slice(subtasks, func(i, j int) bool {
			return subtasks[i].ChildOrder < subtasks[j].ChildOrder
		})
		if len(subtasks) == 0 {
			if hasPrefix(name, cfg.SkipPrefixes) {
				continue
			}
			if cfg.SkipDeadline == "not_overdue" && task.Deadline != nil && task.Deadline.ParsedDate.After(now) {
				continue
			}
			labels := task.Labels
			if len(cfg.IgnoreLabels) > 0 {
				ignore := false
				for _, label := range labels {
					for _, ignoreLabel := range cfg.IgnoreLabels {
						if label == ignoreLabel {
							ignore = true
							break
						}
					}
				}
				if ignore {
					continue
				}
			}
			nextTasks = append(nextTasks, task)
		} else {
			if strings.HasSuffix(name, cfg.SequentialMarker) {
				logger.Debug("Sequential task", "tasks", subtasks, "order", subtasks[len(subtasks)-1].ChildOrder)
				working_on = append(working_on, subtasks[len(subtasks)-1])
			} else {
				working_on = append(working_on, subtasks...)
			}
		}

	}

	return nextTasks
}

