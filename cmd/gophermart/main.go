package main

import (
	"context"
	"errors"
	"github.com/RussiaFPS/gofermart/internal/config"
	"github.com/RussiaFPS/gofermart/internal/handlers"
	"github.com/RussiaFPS/gofermart/internal/logger"
	"github.com/RussiaFPS/gofermart/internal/service"
	"github.com/RussiaFPS/gofermart/internal/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := logger.InitLog()

	cfg, err := config.GetConfig(log)
	if err != nil {
		log.Error(err.Error())
		return
	}

	storages, err := storage.NewStorage(context.Background(), cfg, log)
	if err != nil {
		return
	}
	defer storages.Close()
	services := service.NewService(context.Background(), storages, log, cfg)
	router := handlers.NewRouter(services, log, cfg)

	srv := &http.Server{
		Addr:    cfg.Server,
		Handler: router,
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("listen: %s\n", err)
			return
		}
	}()

	log.Infof("Server started at %v", cfg.Server)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		log.Error("Server Shutdown:", err)
		return
	}
	log.Info("Server exiting")
}
