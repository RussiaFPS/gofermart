package main

import (
	"context"
	"github.com/RussiaFPS/gofermart/internal/config"
	"github.com/RussiaFPS/gofermart/internal/handlers"
	"github.com/RussiaFPS/gofermart/internal/logger"
	"github.com/RussiaFPS/gofermart/internal/service"
	"github.com/RussiaFPS/gofermart/internal/storage"
	"net/http"
	"time"
)

func main() {
	log := logger.InitLog()

	cfg, err := config.GetConfig(log)
	if err != nil {
		log.Error(err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	storages, err := storage.NewStorage(ctx, cfg, log)
	if err != nil {
		return
	}
	services := service.NewService(ctx, storages, log, cfg)
	router := handlers.NewRouter(services, log, cfg)

	err = http.ListenAndServe(cfg.Server, router)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer storages.Close()
}
