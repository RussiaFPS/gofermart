package handlers

import (
	"github.com/RussiaFPS/gofermart/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service service.Services
	log     *logrus.Logger
}

func NewRouter(service service.Services, log *logrus.Logger) chi.Router {
	h := &Handler{
		service: service,
		log:     log,
	}
	router := chi.NewRouter()

	router.Route("/api/user", func(r chi.Router) {
		r.Use(h.gzipHandle)
		r.Post("/register", h.userRegstr)
		r.Post("/login", h.userAuth)
	})

	router.Group(func(r chi.Router) {
		r.Use(h.gzipHandle)
		r.Use(h.checkUserAuth)
		r.Post("/api/user/orders", h.addOrder)
		r.Get("/api/user/orders", h.getOrders)
		r.Get("/api/user/balance", h.getBalance)
		r.Post("/api/user/balance/withdraw", h.withdraw)
		r.Get("/api/user/withdrawals", h.getWithdrawals)
	})

	return router
}
