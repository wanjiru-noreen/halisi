package database

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/routes"
)

func TestConnectInvalidConfiguration(t *testing.T) {
	for _, value := range []string{"", "postgres://user:secret%zz@localhost/db"} {
		pool, err := Connect(context.Background(), value)
		if pool != nil || err == nil {
			t.Fatal("expected configuration error and nil pool")
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("error leaked credentials")
		}
	}
}

func TestConnectTimeout(t *testing.T) {
	// Accept TCP connections without speaking PostgreSQL to exercise the timeout.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() { <-done; conn.Close() }()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	pool, err := Connect(ctx, "postgres://user:secret@"+listener.Addr().String()+"/db?sslmode=disable")
	if pool != nil || err == nil || !strings.Contains(err.Error(), "connection check failed") {
		t.Fatal("expected connection check failure and nil pool")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("error leaked credentials")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("connect did not respect caller deadline")
	}
}

func TestConnectPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("PostgreSQL ping failed")
	}
	if pool.Config().MaxConns != 10 {
		t.Fatal("expected pool limit of 10 connections")
	}
	handler := routes.New(pool)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/db", nil))
	if response.Code != http.StatusOK {
		t.Fatal("database health must return 200 while PostgreSQL is available")
	}
	pool.Close()
	if pool.Ping(ctx) == nil {
		t.Fatal("closed pool should reject Ping")
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/db", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("database health must return 503 after the pool closes")
	}
}
