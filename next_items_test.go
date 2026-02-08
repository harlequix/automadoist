package main

import (
	"testing"
)

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefixes []string
		want     bool
	}{
		{"single match", "*task", []string{"*"}, true},
		{"single no match", "task", []string{"*"}, false},
		{"multiple first matches", "*task", []string{"*", "#"}, true},
		{"multiple second matches", "#task", []string{"*", "#"}, true},
		{"multiple no match", "task", []string{"*", "#"}, false},
		{"empty prefixes", "task", []string{}, false},
		{"empty string", "", []string{"*"}, false},
		{"empty prefix matches all", "task", []string{""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPrefix(tt.s, tt.prefixes); got != tt.want {
				t.Errorf("hasPrefix(%q, %v) = %v, want %v", tt.s, tt.prefixes, got, tt.want)
			}
		})
	}
}
