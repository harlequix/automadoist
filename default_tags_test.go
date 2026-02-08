package main

import (
	"reflect"
	"testing"
)

func TestParseDefaultTags(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        []string
	}{
		{"empty string", "", nil},
		{"no marker", "just a regular description", nil},
		{"single tag", "[automadoist:tags=urgent]", []string{"urgent"}},
		{"multiple tags", "[automadoist:tags=urgent,home,errand]", []string{"urgent", "home", "errand"}},
		{"with surrounding text", "My project\n[automadoist:tags=work,focus]\nMore info", []string{"work", "focus"}},
		{"whitespace in tags", "[automadoist:tags= urgent , home ]", []string{"urgent", "home"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDefaultTags(tt.description)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDefaultTags(%q) = %v, want %v", tt.description, got, tt.want)
			}
		})
	}
}

func TestSetDefaultTagsInDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		tags        []string
		want        string
	}{
		{"add to empty", "", []string{"urgent"}, "[automadoist:tags=urgent]"},
		{"add to existing text", "My project", []string{"urgent", "home"}, "My project\n[automadoist:tags=urgent,home]"},
		{"replace existing", "Info\n[automadoist:tags=old]\nMore", []string{"new1", "new2"}, "Info\n[automadoist:tags=new1,new2]\nMore"},
		{"clear tags with existing marker", "Info\n[automadoist:tags=old]", []string{}, "Info"},
		{"clear tags no marker", "Info", []string{}, "Info"},
		{"add to empty with no tags", "", []string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setDefaultTagsInDescription(tt.description, tt.tags)
			if got != tt.want {
				t.Errorf("setDefaultTagsInDescription(%q, %v) = %q, want %q", tt.description, tt.tags, got, tt.want)
			}
		})
	}
}
