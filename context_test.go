package main

import (
	"reflect"
	"sort"
	"testing"

	"github.com/harlequix/godoist"
)

func TestToSet(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]bool
	}{
		{"empty", nil, map[string]bool{}},
		{"single", []string{"a"}, map[string]bool{"a": true}},
		{"multiple", []string{"a", "b", "c"}, map[string]bool{"a": true, "b": true, "c": true}},
		{"duplicates", []string{"a", "a"}, map[string]bool{"a": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSet(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toSet(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeRetainedLabels(t *testing.T) {
	tests := []struct {
		name         string
		taskLabels   []string
		ignoreLabels []string
		want         []string
	}{
		{"no labels", nil, []string{"waiting"}, []string{}},
		{"no ignore labels", []string{"next", "home"}, nil, []string{}},
		{"retains matching ignore labels", []string{"next", "waiting", "home"}, []string{"waiting", "review"}, []string{"waiting"}},
		{"retains all matching", []string{"waiting", "review"}, []string{"waiting", "review"}, []string{"waiting", "review"}},
		{"no overlap", []string{"next", "home"}, []string{"waiting", "review"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &godoist.Task{Labels: tt.taskLabels}
			got := computeRetainedLabels(task, tt.ignoreLabels)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeRetainedLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeSaveableLabels(t *testing.T) {
	tests := []struct {
		name          string
		taskLabels    []string
		contextLabels []string
		want          []string
	}{
		{"empty task labels", nil, []string{"home", "laptop"}, nil},
		{"no context labels", []string{"home", "next"}, nil, nil},
		{"intersection", []string{"next", "home", "laptop", "waiting"}, []string{"home", "laptop", "desktop"}, []string{"home", "laptop"}},
		{"no overlap", []string{"next", "waiting"}, []string{"home", "laptop"}, nil},
		{"sorted output", []string{"laptop", "home", "next"}, []string{"home", "laptop"}, []string{"home", "laptop"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &godoist.Task{Labels: tt.taskLabels}
			got := computeSaveableLabels(task, tt.contextLabels)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeSaveableLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeExpectedDefaults(t *testing.T) {
	projectTags := map[string][]string{
		"proj1": {"home", "laptop", "extra"},
		"proj2": {"desktop"},
	}
	projectColors := map[string]string{
		"proj1": "red",
		"proj2": "blue",
	}
	colorPriority := map[string]int{
		"red": 3,
	}
	contextLabels := []string{"home", "laptop", "desktop"}

	tests := []struct {
		name         string
		projectID    string
		wantLabels   []string
		wantPriority godoist.PRIORITY_LEVEL
	}{
		{"project with tags and color priority", "proj1", []string{"home", "laptop"}, godoist.MEDIUM},
		{"project with tags no color priority", "proj2", []string{"desktop"}, godoist.VERY_LOW},
		{"unknown project", "proj3", nil, godoist.VERY_LOW},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &godoist.Task{ProjectID: tt.projectID}
			gotLabels, gotPriority := computeExpectedDefaults(task, projectTags, projectColors, colorPriority, contextLabels)
			sort.Strings(gotLabels)
			sort.Strings(tt.wantLabels)
			if !reflect.DeepEqual(gotLabels, tt.wantLabels) {
				t.Errorf("labels = %v, want %v", gotLabels, tt.wantLabels)
			}
			if gotPriority != tt.wantPriority {
				t.Errorf("priority = %v, want %v", gotPriority, tt.wantPriority)
			}
		})
	}
}

func TestHasCustomizations(t *testing.T) {
	contextLabels := []string{"home", "laptop", "desktop"}

	tests := []struct {
		name            string
		taskLabels      []string
		taskPriority    godoist.PRIORITY_LEVEL
		defaultLabels   []string
		defaultPriority godoist.PRIORITY_LEVEL
		want            bool
	}{
		{
			"matches defaults",
			[]string{"next", "home", "laptop"},
			godoist.MEDIUM,
			[]string{"home", "laptop"},
			godoist.MEDIUM,
			false,
		},
		{
			"different priority",
			[]string{"next", "home", "laptop"},
			godoist.HIGH,
			[]string{"home", "laptop"},
			godoist.MEDIUM,
			true,
		},
		{
			"extra context label",
			[]string{"next", "home", "laptop", "desktop"},
			godoist.MEDIUM,
			[]string{"home", "laptop"},
			godoist.MEDIUM,
			true,
		},
		{
			"missing context label",
			[]string{"next", "home"},
			godoist.MEDIUM,
			[]string{"home", "laptop"},
			godoist.MEDIUM,
			true,
		},
		{
			"no defaults no customizations",
			[]string{"next"},
			godoist.VERY_LOW,
			nil,
			godoist.VERY_LOW,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &godoist.Task{Labels: tt.taskLabels, Priority: tt.taskPriority}
			got := hasCustomizations(task, tt.defaultLabels, tt.defaultPriority, contextLabels)
			if got != tt.want {
				t.Errorf("hasCustomizations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildProjectMaps(t *testing.T) {
	projects := []godoist.Project{
		{ID: "p1", Color: "red", Description: "[automadoist:tags=home,laptop]"},
		{ID: "p2", Color: "blue", Description: "no tags here"},
		{ID: "p3", Color: "green", Description: "[automadoist:tags=desktop]"},
	}

	tags, colors := buildProjectMaps(projects)

	// Check tags
	if !reflect.DeepEqual(tags["p1"], []string{"home", "laptop"}) {
		t.Errorf("tags[p1] = %v, want [home laptop]", tags["p1"])
	}
	if _, ok := tags["p2"]; ok {
		t.Errorf("tags[p2] should not exist")
	}
	if !reflect.DeepEqual(tags["p3"], []string{"desktop"}) {
		t.Errorf("tags[p3] = %v, want [desktop]", tags["p3"])
	}

	// Check colors
	if colors["p1"] != "red" {
		t.Errorf("colors[p1] = %v, want red", colors["p1"])
	}
	if colors["p2"] != "blue" {
		t.Errorf("colors[p2] = %v, want blue", colors["p2"])
	}
	if colors["p3"] != "green" {
		t.Errorf("colors[p3] = %v, want green", colors["p3"])
	}
}
