package main

import (
	"sort"

	"github.com/harlequix/godoist"
)

type taskContext struct {
	Labels   []string               `json:"labels"`
	Priority godoist.PRIORITY_LEVEL `json:"priority"`
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// computeRetainedLabels returns only the labels from ignoreLabels that exist on the task.
// These are labels that should be kept when stripping everything else.
func computeRetainedLabels(task *godoist.Task, ignoreLabels []string) []string {
	ignoreSet := toSet(ignoreLabels)
	retained := []string{}
	for _, label := range task.Labels {
		if ignoreSet[label] {
			retained = append(retained, label)
		}
	}
	return retained
}

// computeSaveableLabels returns the intersection of task.Labels with contextLabels.
func computeSaveableLabels(task *godoist.Task, contextLabels []string) []string {
	ctxSet := toSet(contextLabels)
	var saveable []string
	for _, label := range task.Labels {
		if ctxSet[label] {
			saveable = append(saveable, label)
		}
	}
	sort.Strings(saveable)
	return saveable
}

// computeExpectedDefaults returns the labels and priority that defaults would produce for this task.
func computeExpectedDefaults(task *godoist.Task, projectTags map[string][]string, projectColors map[string]string, colorPriority map[string]int, contextLabels []string) ([]string, godoist.PRIORITY_LEVEL) {
	ctxSet := toSet(contextLabels)

	// Expected labels: intersection of project's default tags with contextLabels
	var expectedLabels []string
	if tags, ok := projectTags[task.ProjectID]; ok {
		for _, tag := range tags {
			if ctxSet[tag] {
				expectedLabels = append(expectedLabels, tag)
			}
		}
	}
	sort.Strings(expectedLabels)

	// Expected priority: color priority for project, or VERY_LOW
	expectedPriority := godoist.VERY_LOW
	if color, ok := projectColors[task.ProjectID]; ok {
		if p, ok := colorPriority[color]; ok {
			expectedPriority = godoist.PRIORITY_LEVEL(p)
		}
	}

	return expectedLabels, expectedPriority
}

// hasCustomizations returns true if the task's current context-tracked state differs from defaults.
func hasCustomizations(task *godoist.Task, defaultLabels []string, defaultPriority godoist.PRIORITY_LEVEL, contextLabels []string) bool {
	currentLabels := computeSaveableLabels(task, contextLabels)

	if task.Priority != defaultPriority {
		return true
	}

	if len(currentLabels) != len(defaultLabels) {
		return true
	}
	// Both are sorted, so direct comparison works
	for i := range currentLabels {
		if currentLabels[i] != defaultLabels[i] {
			return true
		}
	}
	return false
}

// saveContext saves the task's current context-tracked labels and priority as a context comment.
func saveContext(task *godoist.Task, contextLabels []string) error {
	labels := computeSaveableLabels(task, contextLabels)
	ctx := map[string]interface{}{
		"labels":   labels,
		"priority": int(task.Priority),
	}
	return task.SetContext(ctx)
}

// restoreContext reads and parses the saved context comment. Returns nil if none exists.
func restoreContext(task *godoist.Task) (*taskContext, error) {
	ctx, err := task.GetContext()
	if err != nil {
		return nil, err
	}
	if len(ctx) == 0 {
		return nil, nil
	}

	tc := &taskContext{
		Priority: godoist.VERY_LOW,
	}

	if labelsRaw, ok := ctx["labels"]; ok {
		if labelsSlice, ok := labelsRaw.([]interface{}); ok {
			for _, l := range labelsSlice {
				if s, ok := l.(string); ok {
					tc.Labels = append(tc.Labels, s)
				}
			}
		}
	}

	if priorityRaw, ok := ctx["priority"]; ok {
		switch p := priorityRaw.(type) {
		case float64:
			tc.Priority = godoist.PRIORITY_LEVEL(int(p))
		case int:
			tc.Priority = godoist.PRIORITY_LEVEL(p)
		}
	}

	return tc, nil
}

// buildProjectMaps precomputes project tags and color lookup maps from a project list.
func buildProjectMaps(projects []godoist.Project) (map[string][]string, map[string]string) {
	projectTags := buildProjectTagsMap(projects)
	projectColors := make(map[string]string, len(projects))
	for _, p := range projects {
		projectColors[p.ID] = p.Color
	}
	return projectTags, projectColors
}
