package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/harlequix/godoist"
)

type DefaultTagsConfig struct {
	AvailableTags []string `koanf:"available_tags"`
}

var defaultTagsRegex = regexp.MustCompile(`\[automadoist:tags=([^\]]+)\]`)

func parseDefaultTags(description string) []string {
	match := defaultTagsRegex.FindStringSubmatch(description)
	if match == nil {
		return nil
	}
	raw := strings.Split(match[1], ",")
	tags := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func setDefaultTagsInDescription(description string, tags []string) string {
	marker := ""
	if len(tags) > 0 {
		marker = "[automadoist:tags=" + strings.Join(tags, ",") + "]"
	}

	if defaultTagsRegex.MatchString(description) {
		replaced := defaultTagsRegex.ReplaceAllString(description, marker)
		return strings.TrimSpace(replaced)
	}

	if marker == "" {
		return description
	}

	if description == "" {
		return marker
	}
	return description + "\n" + marker
}

func applyDefaultTags(client *godoist.Todoist, nextTasks []*godoist.Task, projects []godoist.Project) {
	projectTags := make(map[string][]string)
	for _, p := range projects {
		tags := parseDefaultTags(p.Description)
		if len(tags) > 0 {
			projectTags[p.ID] = tags
		}
	}
	if len(projectTags) == 0 {
		return
	}
	for _, task := range nextTasks {
		tags, ok := projectTags[task.ProjectID]
		if !ok {
			continue
		}
		for _, tag := range tags {
			task.AddLabel(tag)
		}
	}
}

// sortProjectsByOrder sorts projects into a stable tree order matching
// Todoist's sidebar: depth-first traversal ordered by ChildOrder at each level.
func sortProjectsByOrder(projects []*godoist.Project) {
	childrenOf := make(map[string][]*godoist.Project)
	for _, p := range projects {
		childrenOf[p.ParentID] = append(childrenOf[p.ParentID], p)
	}
	for _, children := range childrenOf {
		sort.Slice(children, func(i, j int) bool {
			return children[i].ChildOrder < children[j].ChildOrder
		})
	}

	ordered := make([]*godoist.Project, 0, len(projects))
	var walk func(parentID string)
	walk = func(parentID string) {
		for _, p := range childrenOf[parentID] {
			ordered = append(ordered, p)
			walk(p.ID)
		}
	}
	// Root projects have ParentID "" (or possibly "0" depending on API)
	walk("")
	// Include any projects not reached (e.g. different root sentinel)
	if len(ordered) < len(projects) {
		seen := make(map[string]bool, len(ordered))
		for _, p := range ordered {
			seen[p.ID] = true
		}
		for _, p := range projects {
			if !seen[p.ID] {
				ordered = append(ordered, p)
			}
		}
	}
	copy(projects, ordered)
}

func defaultTagsCommand(client *godoist.Todoist, cfg DefaultTagsConfig) error {
	if len(cfg.AvailableTags) == 0 {
		return fmt.Errorf("no available tags configured; set default_tags.available_tags in config")
	}

	allProjects := client.Projects.All()
	if len(allProjects) == 0 {
		return fmt.Errorf("no projects found")
	}

	sortProjectsByOrder(allProjects)

	for {
		options := make([]huh.Option[string], 0, len(allProjects))
		for _, p := range allProjects {
			label := p.Name
			tags := parseDefaultTags(p.Description)
			if len(tags) > 0 {
				label += " [" + strings.Join(tags, ", ") + "]"
			}
			options = append(options, huh.NewOption(label, p.ID))
		}

		var projectID string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a project").
					Options(options...).
					Value(&projectID),
			),
		).Run()
		if err != nil {
			return nil
		}

		project := client.Projects.Get(projectID)
		if project == nil {
			return fmt.Errorf("project not found: %s", projectID)
		}

		currentTags := parseDefaultTags(project.Description)
		currentSet := make(map[string]bool, len(currentTags))
		for _, t := range currentTags {
			currentSet[t] = true
		}

		tagOptions := make([]huh.Option[string], 0, len(cfg.AvailableTags))
		for _, tag := range cfg.AvailableTags {
			tagOptions = append(tagOptions, huh.NewOption(tag, tag).Selected(currentSet[tag]))
		}

		var selectedTags []string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Default tags for " + project.Name).
					Options(tagOptions...).
					Value(&selectedTags),
			),
		).Run()
		if err != nil {
			return nil
		}

		newDescription := setDefaultTagsInDescription(project.Description, selectedTags)
		project.Update("description", newDescription)
		client.Commit()

		if len(selectedTags) > 0 {
			fmt.Printf("Set default tags for %q: %s\n", project.Name, strings.Join(selectedTags, ", "))
		} else {
			fmt.Printf("Cleared default tags for %q\n", project.Name)
		}
	}
}
