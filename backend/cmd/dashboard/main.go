// Command dashboard runs the golangci-lint web dashboard's API as a
// standalone binary — separate process, separate port, separate DB and
// Redis from the main rt-external-api server. See golangci/plans/ for the
// design suite and golangci/plans/2026-08-04-golangci-m{1,2,3}-implementation.md
// for why.
//
// Scope so far: M1's storage/config/RBAC skeleton, M2's scan loop, M3's
// plan loop, M4's approve + native-fix-only apply + rescan, plus
// (2026-08-05) rollbacks (git revert only, never reset/force-push).
// history/ui and a real AI Layer client are still not built.
//
// @title           Golangci Lint Dashboard API
// @version         1.0
// @description     Scan -> Plan -> Approve -> Fix -> Rollback REST API for golangci-lint issue triage.
// @BasePath        /api/v1
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        X-API-Key
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golangci/backend/api"
	"golangci/backend/planner"
	"golangci/backend/storage"
	"golangci/backend/worker"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err) // logger itself failed to construct — nothing else can log this
	}
	defer func() { _ = logger.Sync() }()

	_ = godotenv.Load() // optional — env vars may already be set by the environment

	if err := api.LoadPermissions("backend/config/permissions.json"); err != nil {
		logger.Fatal("failed to load backend/config/permissions.json", zap.Error(err))
	}

	db, err := storage.Connect()
	if err != nil {
		logger.Fatal("failed to connect to GOLANGCI_MYSQL_DB_*", zap.Error(err))
	}

	if err := storage.Migrate(db); err != nil {
		logger.Fatal("failed to migrate golangci dashboard schema", zap.Error(err))
	}

	severityMap, defaultSeverity, err := worker.LoadSeverityMap(
		"backend/config/severity-mapping.json",
	)
	if err != nil {
		logger.Fatal("failed to load backend/config/severity-mapping.json", zap.Error(err))
	}

	redisDB, err := strconv.Atoi(getEnvDefault("GOLANGCI_REDIS_DB", "1"))
	if err != nil {
		logger.Fatal("invalid GOLANGCI_REDIS_DB", zap.Error(err))
	}
	lock := worker.NewLock(
		getEnvDefault("GOLANGCI_REDIS_ADDR", "localhost:6379"),
		os.Getenv("GOLANGCI_REDIS_PASSWORD"),
		redisDB,
	)

	plannerSvc := planner.NewService(db, planner.NewMockClient())

	w := worker.NewWorker(lock.Client(), lock, db, logger, severityMap, defaultSeverity, plannerSvc)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	var workerWG sync.WaitGroup
	workerWG.Add(4)
	go func() {
		defer workerWG.Done()
		w.RunScans(workerCtx)
	}()
	go func() {
		defer workerWG.Done()
		w.RunPlans(workerCtx)
	}()
	go func() {
		defer workerWG.Done()
		w.RunFixes(workerCtx)
	}()
	go func() {
		defer workerWG.Done()
		w.RunRollbacks(workerCtx)
	}()

	port := getEnvDefault("GOLANGCI_DASHBOARD_PORT", "8081")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: api.NewRouter(db, lock.Client(), lock, plannerSvc),
	}

	go func() {
		logger.Info("golangci dashboard listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("dashboard server stopped unexpectedly", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down golangci dashboard")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful HTTP shutdown failed", zap.Error(err))
	}

	cancelWorker()
	waitWithTimeout(&workerWG, 10*time.Second, logger)
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration, logger *zap.Logger) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		logger.Info("worker stopped cleanly")
	case <-time.After(timeout):
		logger.Warn("worker did not stop within the shutdown grace period")
	}
}
