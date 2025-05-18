package main

import (
	"github.com/fokv/cron"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test homeHandler
func TestHomeHandler(t *testing.T) {
	t.Run("returns home page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		homeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		expected := "Home Page"
		if !strings.Contains(rr.Body.String(), expected) {
			t.Errorf("expected body to contain %q, got %q", expected, rr.Body.String())
		}
	})
}

// Test listScheduler
func TestListScheduler(t *testing.T) {
	// Setup test scheduler
	originalGCron := gcron
	defer func() { gcron = originalGCron }()

	testScheduler := cron.NewDynamicScheduler("TestScheduler")
	testScheduler.Start()
	defer testScheduler.Stop()

	// Register test function
	err := testScheduler.RegisterFunc(cron.NamedFunc{
		Name: "test_job",
		Spec: "@every 1s",
	})
	if err != nil {
		t.Fatal(err)
	}

	gcron = testScheduler

	t.Run("returns scheduler status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/list", nil)
		rr := httptest.NewRecorder()

		listScheduler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		expected := []string{"TestScheduler", "test_job", "@every 1s"}
		body := rr.Body.String()
		for _, s := range expected {
			if !strings.Contains(body, s) {
				t.Errorf("expected body to contain %q", s)
			}
		}
	})
}
