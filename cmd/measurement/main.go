package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yaelmoshi/netzbremse/internal/collector"
	"github.com/yaelmoshi/netzbremse/internal/config"
	"github.com/yaelmoshi/netzbremse/internal/postgres"
)

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbConfig, err := config.LoadDatabase()
	if err != nil {
		log.Fatal(err)
	}
	measurementConfig, err := config.LoadMeasurement()
	if err != nil {
		log.Fatal(err)
	}

	store, err := postgres.New(ctx, dbConfig.URI)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	if err := store.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	if measurementConfig.ImportDir != "" {
		imported, err := store.ImportDir(ctx, measurementConfig.ImportDir)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("imported %d JSON measurements from %s", imported, measurementConfig.ImportDir)
	}

	runCollection := func(reason string) {
		item, err := collector.Run(ctx, measurementConfig)
		if err != nil {
			log.Printf("collection failed (%s): %v", reason, err)
			return
		}
		if err := store.Insert(ctx, item); err != nil {
			log.Printf("persist failed (%s): %v", reason, err)
			return
		}
		log.Printf(
			"stored measurement (%s): endpoint=%s success=%t down=%.0f up=%.0f latency=%.2f",
			reason,
			item.Endpoint,
			item.Success,
			deref(item.DownloadBPS),
			deref(item.UploadBPS),
			deref(item.LatencyMS),
		)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	server := &http.Server{
		Addr:              measurementConfig.ListenAddress,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("measurement worker listening on %s", measurementConfig.ListenAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	ticker := time.NewTicker(measurementConfig.Interval)
	defer ticker.Stop()

	log.Printf("measurement loop armed for endpoint %s", measurementConfig.Endpoint)
	runCollection("startup")

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			_ = server.Shutdown(shutdownCtx)
			return
		case <-ticker.C:
			runCollection("schedule")
		}
	}
}
