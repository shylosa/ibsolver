package application

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

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

func TestNumericalCalculationRestartAndFailurePreservesResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numerical.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	state, err := service.Load()
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.Calculate(context.Background(), state.Active.ID)
	if err != nil || state.Run == nil || state.Run.ResultVersion != domain.NumericalVersion || state.Stale {
		t.Fatalf("calculate: %+v %v", state, err)
	}
	previous := *state.Run
	c := state.Active
	n := *c.Numerical
	n.Model.Velocity = 0.5
	n.Solver.RelTolerance = 1e-8
	c.Numerical = &n
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil || !state.Stale || *state.Run.NumericalSnapshot == n {
		t.Fatal("snapshot changed with inputs", err)
	}
	store.Close()
	store, err = sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service = New(store)
	state, err = service.Load()
	if err != nil || !state.Stale || !reflect.DeepEqual(*state.Run, previous) || *state.Active.Numerical != n {
		t.Fatalf("restart lost numeric data/snapshot/settings: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = service.Calculate(ctx, state.Active.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if _, err = service.Calculate(context.Background(), state.Active.ID+100); err == nil {
		t.Fatal("accepted wrong configuration")
	}
	n.Solver.MaxSteps = 1
	c.Numerical = &n
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Calculate(context.Background(), state.Active.ID); err == nil {
		t.Fatal("expected step limit")
	}
	state, err = service.Load()
	if err != nil || !reflect.DeepEqual(*state.Run, previous) {
		t.Fatal("failed calculation replaced previous result", err)
	}
	n.Model.Mass = -1
	if _, err = service.SaveConfiguration(c, state.Settings); err == nil {
		t.Fatal("accepted invalid model")
	}
	state, err = service.Load()
	if err != nil || !reflect.DeepEqual(*state.Run, previous) {
		t.Fatal("invalid input replaced previous result")
	}
}

func TestCancelActiveCalculationDoesNotBlockLoad(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)
	state, err := service.Load()
	if err != nil {
		t.Fatal(err)
	}
	c := state.Active
	n := *c.Numerical
	n.Model.Duration, n.Model.Samples = 10000, 2
	n.Solver.InitialStep, n.Solver.MaxStep, n.Solver.MaxSteps = 1e-5, 1e-5, 1000000
	c.Numerical = &n
	state, err = service.SaveConfiguration(c, state.Settings)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := service.Calculate(context.Background(), state.Active.ID); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		service.mu.Lock()
		running := service.cancel != nil
		service.mu.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("calculation did not start")
		}
		runtime.Gosched()
	}
	if _, err := service.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Calculate(context.Background(), state.Active.ID); err == nil {
		t.Fatal("allowed concurrent calculation")
	}
	service.CancelCalculation()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected active cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("active cancellation timed out")
	}
}
