package main

import (
	"sort"
	"strings"
	"time"

	"github.com/harlequix/godoist"
)

type NextItemsConfig struct {
	EntryPoint       string   `koanf:"entry_point"`
	SkipPrefixes     []string `koanf:"skip_prefixes"`
	Recursive        bool     `koanf:"recursive"`
	SequentialMarker string   `koanf:"sequential_marker"`
	SkipDeadline     string   `koanf:"skip_deadline"`
	ManagedLabels    []string `koanf:"managed_labels"`
	IgnoreLabels     []string `koanf:"ignore_labels"`
	Prune            bool     `koanf:"prune"`
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

	for _, t := range needAddition {
		logger.Debug("Add label to task", "task", t.Content, "label", cfg.ManagedLabels[0])
		t.AddLabel(cfg.ManagedLabels[0])
	}

	for _, t := range needRemoval {
		logger.Debug("Remove label from task", "task", t.Content, "label", cfg.ManagedLabels[0])
		t.RemoveLabel(cfg.ManagedLabels[0])
	}

	client.API.Commit()

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
	logger.Debug("Number of task in project", "project", project.Name, "#", len(tasks))
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
			return subtasks[i].Order < subtasks[j].Order
		})
		if len(subtasks) == 0 {
			if len(cfg.SkipPrefixes) > 0 && strings.HasPrefix(name, cfg.SkipPrefixes[0]) {
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
				logger.Debug("Sequential task", "tasks", subtasks, "order", subtasks[len(subtasks)-1].Order)
				working_on = append(working_on, subtasks[len(subtasks)-1])
			} else {
				working_on = append(working_on, subtasks...)
			}
		}

	}

	return nextTasks
}
