package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type NamedFunc struct {
	ID          cron.EntryID  `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Spec        string        `json:"spec"`
	Func        func()        `json:"-"`
	Timeout     time.Duration `json:"timeout"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

func (nf NamedFunc) String() string {
	indent, marshalErr := json.MarshalIndent(nf, "", "\t")
	if marshalErr != nil {
		fmt.Fprint(os.Stderr, "Marshal NamedFunc error:", marshalErr)
		return ""
	}
	return string(indent)
}

type DynamicScheduler struct {
	Name  string               `json:"name"`
	Cron  *cron.Cron           `json:"-"`
	Funcs map[string]NamedFunc `json:"funcs"`
	Mu    sync.RWMutex         `json:"-"`
}

func (ds DynamicScheduler) String() string {
	indent, marshalErr := json.MarshalIndent(ds, "", "\t")
	if marshalErr != nil {
		fmt.Fprint(os.Stderr, "Marshal NamedFunc error:", marshalErr)
		return ""
	}
	return string(indent)
}

func NewDynamicScheduler(name string) *DynamicScheduler {
	return &DynamicScheduler{
		Name:  name,
		Cron:  cron.New(cron.WithSeconds()),
		Funcs: make(map[string]NamedFunc),
	}
}

func (d *DynamicScheduler) Start() {
	d.Cron.Start()
}

func (d *DynamicScheduler) Stop() {
	d.Cron.Stop()
}

func (d *DynamicScheduler) RegisterFunc(nf NamedFunc) error {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	if _, exists := d.Funcs[nf.Name]; exists {
		return fmt.Errorf("function %q already registered", nf.Name)
	}

	wrapped := d.wrapFunction(nf.Func, nf.Timeout)
	entryID, err := d.Cron.AddFunc(nf.Spec, wrapped)
	if err != nil {
		return fmt.Errorf("invalid cron spec: %w", err)
	}

	nf.ID = entryID
	nf.UpdatedAt = time.Now()
	d.Funcs[nf.Name] = nf
	return nil
}

func (d *DynamicScheduler) UpdateSpec(name, newSpec string) error {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	nf, exists := d.Funcs[name]
	if !exists {
		return fmt.Errorf("function %q not found", name)
	}

	// Remove old entry and create new
	d.Cron.Remove(nf.ID)
	wrapped := d.wrapFunction(nf.Func, nf.Timeout)

	entryID, err := d.Cron.AddFunc(newSpec, wrapped)
	if err != nil {
		return fmt.Errorf("invalid new spec: %w", err)
	}

	nf.ID = entryID
	nf.Spec = newSpec
	nf.UpdatedAt = time.Now()
	d.Funcs[name] = nf
	return nil
}

func (d *DynamicScheduler) wrapFunction(fn func(), timeout time.Duration) func() {
	return func() {
		if timeout <= 0 {
			fn()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			fn()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			// Handle timeout logic
		}
	}
}

func (d *DynamicScheduler) GetFunc(name string) (NamedFunc, bool) {
	d.Mu.RLock()
	defer d.Mu.RUnlock()

	nf, exists := d.Funcs[name]
	return nf, exists
}

func (d *DynamicScheduler) ListFuncs() []NamedFunc {
	d.Mu.RLock()
	defer d.Mu.RUnlock()

	list := make([]NamedFunc, 0, len(d.Funcs))
	for _, nf := range d.Funcs {
		list = append(list, nf)
	}
	return list
}

// MarshalJSON custom implementation
func (d *DynamicScheduler) MarshalJSON() ([]byte, error) {
	type Alias DynamicScheduler
	return json.Marshal(&struct {
		*Alias
		Funcs []NamedFunc `json:"funcs"`
	}{
		Alias: (*Alias)(d),
		Funcs: d.ListFuncs(),
	})
}
