package cron

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartStop(t *testing.T) {
	ds := NewDynamicScheduler("test")
	ds.Start()
	ds.Stop()
	// Test passes if Start/Stop do not panic
}

func TestRegisterFunc_Success(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	err := ds.RegisterFunc(nf)
	require.NoError(t, err)

	registered, exists := ds.GetFunc("test_func")
	require.True(t, exists)
	assert.Equal(t, nf.Name, registered.Name)
	assert.NotZero(t, registered.ID)
	assert.NotZero(t, registered.UpdatedAt)
}

func TestRegisterFunc_Duplicate(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	require.NoError(t, ds.RegisterFunc(nf))

	err := ds.RegisterFunc(nf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegisterFunc_InvalidSpec(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "invalid-spec",
	}
	err := ds.RegisterFunc(nf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron spec")
}

func TestUpdateSpec_Success(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	require.NoError(t, ds.RegisterFunc(nf))

	oldEntry, _ := ds.GetFunc("test_func")

	newSpec := "*/2 * * * * *"
	require.NoError(t, ds.UpdateSpec("test_func", newSpec))

	updated, exists := ds.GetFunc("test_func")
	require.True(t, exists)
	assert.Equal(t, newSpec, updated.Spec)
	assert.NotEqual(t, oldEntry.ID, updated.ID)
}

func TestUpdateSpec_InvalidSpec(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	require.NoError(t, ds.RegisterFunc(nf))

	err := ds.UpdateSpec("test_func", "invalid-spec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid new spec")

	// Ensure original spec remains
	current, _ := ds.GetFunc("test_func")
	assert.Equal(t, nf.Spec, current.Spec)
}

func TestUpdateSpec_NotFound(t *testing.T) {
	ds := NewDynamicScheduler("test")
	err := ds.UpdateSpec("nonexistent", "* * * * * *")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetFunc(t *testing.T) {
	ds := NewDynamicScheduler("test")
	_, exists := ds.GetFunc("nonexistent")
	assert.False(t, exists)

	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	require.NoError(t, ds.RegisterFunc(nf))

	registered, exists := ds.GetFunc("test_func")
	require.True(t, exists)
	assert.Equal(t, nf.Name, registered.Name)
}

func TestListFuncs(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf1 := NamedFunc{Name: "func1", Func: func() {}, Spec: "* * * * * *"}
	nf2 := NamedFunc{Name: "func2", Func: func() {}, Spec: "* * * * * *"}
	require.NoError(t, ds.RegisterFunc(nf1))
	require.NoError(t, ds.RegisterFunc(nf2))

	list := ds.ListFuncs()
	assert.Len(t, list, 2)
	names := map[string]bool{
		list[0].Name: true,
		list[1].Name: true,
	}
	assert.True(t, names["func1"])
	assert.True(t, names["func2"])
}

func TestMarshalJSON(t *testing.T) {
	ds := NewDynamicScheduler("test")
	nf := NamedFunc{
		Name: "test_func",
		Func: func() {},
		Spec: "* * * * * *",
	}
	require.NoError(t, ds.RegisterFunc(nf))

	data, err := json.Marshal(ds)
	require.NoError(t, err)

	var result struct {
		Name  string      `json:"name"`
		Funcs []NamedFunc `json:"funcs"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "test", result.Name)
	require.Len(t, result.Funcs, 1)
	assert.Equal(t, "test_func", result.Funcs[0].Name)
}

func TestWrapFunction_Timeout(t *testing.T) {
	ds := NewDynamicScheduler("test")
	timeout := 100 * time.Millisecond
	block := make(chan struct{})
	completed := make(chan struct{})

	fn := func() {
		<-block
		close(completed)
	}

	wrapped := ds.wrapFunction(fn, timeout)
	go wrapped()

	select {
	case <-completed:
		t.Fatal("function completed unexpectedly")
	case <-time.After(timeout + 50*time.Millisecond):
		// Timeout occurred as expected
	}

	close(block) // Cleanup the blocking goroutine
}

func TestWrapFunction_NoTimeout(t *testing.T) {
	ds := NewDynamicScheduler("test")
	done := make(chan struct{})

	fn := func() {
		close(done)
	}

	wrapped := ds.wrapFunction(fn, time.Second)
	wrapped()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("function did not complete")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ds := NewDynamicScheduler("test")
	var wg sync.WaitGroup
	wg.Add(2)

	// Register functions concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("func%d", i)
			nf := NamedFunc{
				Name: name,
				Func: func() {},
				Spec: "* * * * * *",
			}
			_ = ds.RegisterFunc(nf) // Ignore errors for test purposes
		}
	}()

	// Update functions concurrently
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("func%d", i)
			_ = ds.UpdateSpec(name, "*/2 * * * * *") // Ignore errors
		}
	}()

	wg.Wait()
	// Test passes if no race conditions are detected (run with -race)
}

func TestZeroTimeout(t *testing.T) {
	ds := NewDynamicScheduler("test")
	done := make(chan struct{})

	nf := NamedFunc{
		Name:    "test",
		Func:    func() { close(done) },
		Spec:    "* * * * * *",
		Timeout: 0,
	}

	wrapped := ds.wrapFunction(nf.Func, nf.Timeout)
	go wrapped()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("function with zero timeout did not complete")
	}
}

func TestNegativeTimeout(t *testing.T) {
	ds := NewDynamicScheduler("test")
	done := make(chan struct{})

	nf := NamedFunc{
		Name:    "test",
		Func:    func() { close(done) },
		Spec:    "* * * * * *",
		Timeout: -time.Second,
	}

	wrapped := ds.wrapFunction(nf.Func, nf.Timeout)
	go wrapped()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("function with negative timeout did not complete")
	}
}
