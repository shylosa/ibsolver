package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestVersionFormat(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+$`).MatchString(Version) {
		t.Fatalf("Version = %q, want MAJOR.MINOR (e.g. \"1.0\")", Version)
	}
}

func TestWindowTitleContainsVersion(t *testing.T) {
	main, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(main), `Title:     "IBSolver " + Version,`) {
		t.Fatal(`main.go must set the window title to "IBSolver " + Version`)
	}
}
