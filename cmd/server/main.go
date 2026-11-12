package main

import (
	"log"
	"net/http"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/config"
	"github.com/wanjiru-noreen/halisi/internal/routes"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.New(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("HTTP server listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
