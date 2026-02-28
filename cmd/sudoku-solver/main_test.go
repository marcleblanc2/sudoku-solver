package main

import (
	"testing"
)

func TestResolveInput(t *testing.T) {
	tests := []struct {
		name      string
		flag      string
		args      []string
		want      string
		wantError bool
	}{
		{
			name: "positional arg",
			args: []string{"puzzle.png"},
			want: "puzzle.png",
		},
		{
			name: "flag",
			flag: "puzzle.png",
			want: "puzzle.png",
		},
		{
			name:      "both flag and positional",
			flag:      "a.png",
			args:      []string{"b.png"},
			wantError: true,
		},
		{
			name:      "no input",
			wantError: true,
		},
		{
			name:      "extra positional args",
			args:      []string{"a.png", "b.png"},
			wantError: true,
		},
		{
			name:      "extra args with flag",
			flag:      "a.png",
			args:      []string{"b.png", "c.png"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveInput(tt.flag, tt.args)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
