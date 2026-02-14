package main

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    slog.Level
		wantErr bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"DEBUG", slog.LevelDebug, false},
		{"invalid", slog.Level(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfigVerify(t *testing.T) {
	t.Run("empty token", func(t *testing.T) {
		cfg := config{Token: ""}
		if err := cfg.Verify(); err == nil {
			t.Error("expected error for empty token")
		}
	})
	t.Run("valid token with defaults", func(t *testing.T) {
		cfg := config{Token: "abc123", NextItems: defaultNextItemsConfig(), ReviewsConfig: defaultReviewsConfig(NextItemsConfig{})}
		if err := cfg.Verify(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("empty managed labels", func(t *testing.T) {
		cfg := config{Token: "abc123", NextItems: NextItemsConfig{EntryPoint: "projects"}, ReviewsConfig: defaultReviewsConfig(NextItemsConfig{})}
		if err := cfg.Verify(); err == nil {
			t.Error("expected error for empty managed_labels")
		}
	})
	t.Run("empty review label", func(t *testing.T) {
		cfg := config{Token: "abc123", NextItems: defaultNextItemsConfig(), ReviewsConfig: ReviewsConfig{Label: ""}}
		if err := cfg.Verify(); err == nil {
			t.Error("expected error for empty review label")
		}
	})
}
