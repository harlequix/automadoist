package main

import (
	"testing"
)

func TestPrepare(t *testing.T) {
	t.Run("removes matching skip prefix", func(t *testing.T) {
		reviewCfg := ReviewsConfig{
			Prefixes: []string{"*"},
			Label:    "review",
		}
		nextCfg := NextItemsConfig{
			SkipPrefixes: []string{"*", "#"},
			IgnoreLabels: []string{"waiting", "review"},
		}
		got := prepare(reviewCfg, nextCfg)

		for _, p := range got.SkipPrefixes {
			if p == "*" {
				t.Error("expected '*' to be removed from SkipPrefixes")
			}
		}
		if len(got.SkipPrefixes) != 1 || got.SkipPrefixes[0] != "#" {
			t.Errorf("SkipPrefixes = %v, want [#]", got.SkipPrefixes)
		}
	})

	t.Run("removes matching ignore label", func(t *testing.T) {
		reviewCfg := ReviewsConfig{
			Prefixes: []string{},
			Label:    "review",
		}
		nextCfg := NextItemsConfig{
			SkipPrefixes: []string{"*"},
			IgnoreLabels: []string{"waiting", "review"},
		}
		got := prepare(reviewCfg, nextCfg)

		for _, l := range got.IgnoreLabels {
			if l == "review" {
				t.Error("expected 'review' to be removed from IgnoreLabels")
			}
		}
		if len(got.IgnoreLabels) != 1 || got.IgnoreLabels[0] != "waiting" {
			t.Errorf("IgnoreLabels = %v, want [waiting]", got.IgnoreLabels)
		}
	})

	t.Run("does not modify original config", func(t *testing.T) {
		reviewCfg := ReviewsConfig{
			Prefixes: []string{"*"},
			Label:    "review",
		}
		nextCfg := NextItemsConfig{
			SkipPrefixes: []string{"*", "#"},
			IgnoreLabels: []string{"waiting", "review"},
		}
		prepare(reviewCfg, nextCfg)

		if len(nextCfg.SkipPrefixes) != 2 {
			t.Errorf("original SkipPrefixes was modified: %v", nextCfg.SkipPrefixes)
		}
		if len(nextCfg.IgnoreLabels) != 2 {
			t.Errorf("original IgnoreLabels was modified: %v", nextCfg.IgnoreLabels)
		}
	})

	t.Run("no matching prefix", func(t *testing.T) {
		reviewCfg := ReviewsConfig{
			Prefixes: []string{"@"},
			Label:    "someother",
		}
		nextCfg := NextItemsConfig{
			SkipPrefixes: []string{"*"},
			IgnoreLabels: []string{"waiting"},
		}
		got := prepare(reviewCfg, nextCfg)

		if len(got.SkipPrefixes) != 1 || got.SkipPrefixes[0] != "*" {
			t.Errorf("SkipPrefixes = %v, want [*]", got.SkipPrefixes)
		}
		if len(got.IgnoreLabels) != 1 || got.IgnoreLabels[0] != "waiting" {
			t.Errorf("IgnoreLabels = %v, want [waiting]", got.IgnoreLabels)
		}
	})
}
