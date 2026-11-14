package application

import (
	"path/filepath"
	"testing"

	"ibsolver/internal/domain"
	"ibsolver/internal/storage/sqlite"
)

func TestLoadNeverGeneratesDemo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	state, err := service.Load()
	if err != nil || state.Run != nil || len(state.Configurations) != 1 {
		t.Fatalf("initial state=%+v err=%v", state, err)
	}
	if _, err = service.CreateDemo(state.Active.ID + 100); err == nil {
		t.Fatal("accepted wrong configuration")
	}
	state, err = service.CreateDemo(state.Active.ID)
	if err != nil || state.Run == nil {
		t.Fatalf("demo: %v", err)
	}
	runID := state.Run.ID
	changed := state.Active
	changed.Inputs.Frequency = 2
	state, err = service.SaveConfiguration(changed, state.Settings)
	if err != nil || !state.Stale {
		t.Fatalf("edit: %v", err)
	}
	store.Close()
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	state, err = New(store).Load()
	if err != nil || state.Run == nil || state.Run.ID != runID || !state.Stale || state.Active.Inputs != changed.Inputs {
		t.Fatalf("restart: %+v %v", state, err)
	}
	if state.Run.ResultVersion != domain.ResultVersion {
		t.Fatal("wrong result version")
	}
}
