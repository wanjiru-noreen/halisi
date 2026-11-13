package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/config"
	"github.com/wanjiru-noreen/halisi/internal/database"
	"github.com/wanjiru-noreen/halisi/internal/email"
	"github.com/wanjiru-noreen/halisi/internal/handlers"
	"github.com/wanjiru-noreen/halisi/internal/routes"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	identity, err := config.LoadIdentity(cfg.AppEnv)
	if err != nil {
		return err
	}
	var options []services.AuthOption
	if identity.SMTPHost != "" {
		options = append(options, services.WithVerification(&email.SMTP{Config: identity}, identity.PublicBaseURL))
	}
	if identity.GoogleClientID != "" {
		providerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		provider, err := services.NewGoogleOIDC(providerCtx, identity.GoogleClientID, identity.GoogleClientSecret, identity.PublicBaseURL)
		cancel()
		if err != nil {
			return err
		}
		options = append(options, services.WithGoogle(provider))
	}
	authService, err := services.NewAuth(pool, options...)
	if err != nil {
		return errors.New("could not initialize authentication")
	}
	auth := handlers.NewAuth(authService, cfg.AppEnv)
	resources := handlers.NewResources(pool, auth)
	storefront := handlers.NewStorefront(pool)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes.New(pool, auth.Routes, resources.Routes, storefront.Routes),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	log.Printf("HTTP server listening on %s", server.Addr)
	go func() { serverErrors <- server.ListenAndServe() }()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("HTTP shutdown: %w", err)
		}
		return nil
	}
}
