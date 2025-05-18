package main

import (
	"fmt"
	"github.com/fokv/cron"
	"log"
	"net/http"
	"time"
)

var (
	gcron *cron.DynamicScheduler
)

func init() {
	gcron = cron.NewDynamicScheduler("TestScheduler")
	gcron.Start()
	defer gcron.Stop()

	// Register named function
	err := gcron.RegisterFunc(cron.NamedFunc{
		Name:        "health_check",
		Description: "Service monitoring",
		Spec:        "@every 1s",
		Timeout:     5 * time.Second,
		Func:        func() { fmt.Println("Checking services... =>:Time:", time.Now().Format(time.RFC3339)) },
	})
	if err != nil {
		log.Fatal("Registration failed:", err)
	}
	log.Println("Scheduler before change:", gcron)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Home Page"))
}
func listScheduler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Scheduler: " + gcron.String()))
}
func updateSpec(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("name")
	newSpec := r.URL.Query().Get("newSpec")
	gcron.UpdateSpec(id, newSpec)

	w.Write([]byte("Scheduler: " + gcron.String()))
}

func main() {
	mux := http.NewServeMux()

	// Route handlers
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/list", listScheduler)
	mux.HandleFunc("/update", updateSpec)

	// Start server
	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}

	fmt.Println("Server started at http://localhost:8000")
	server.ListenAndServe()
}
