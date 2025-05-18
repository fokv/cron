package main

import (
	"fmt"
	"github.com/fokv/cron"
	"log"
	"time"
)

func main() {
	// Create scheduler instance
	gcron := cron.NewDynamicScheduler("production")
	// Start scheduler
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
	time.Sleep(5 * time.Second)

	//Scheduler update should be in some other goroutine.
	go func() {
		if err := gcron.UpdateSpec("health_check", "@every 8s"); err != nil {
			log.Println("Update failed:", err)
		}
	}()
	log.Println("Scheduler after change:", gcron)

	//
	time.Sleep(33 * time.Second)
	go func() {
		if err := gcron.UpdateSpec("health_check", "@every 2s"); err != nil {
			log.Println("Update failed:", err)
		}
	}()
	log.Println("Scheduler health_check changed again:", gcron)

	// Keep program running
	select {}
}
