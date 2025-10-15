package handlers

import (
	"github.com/RussiaFPS/gofermart/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type server struct {
	service service.Services
	log     *logrus.Logger
}

func NewRouter(service service.Services, log *logrus.Logger) chi.Router {
	srv := &server{
		service: service,
		log:     log,
	}
	router := chi.NewRouter()

	router.Route("/api/user", func(r chi.Router) {
		r.Use(gzipHandle)
		r.Post("/register", srv.userRegstr)
		r.Post("/login", srv.userAuth)
	})

	router.Group(func(r chi.Router) {
		r.Use(gzipHandle)
		r.Use(srv.checkUserAuth)
		r.Post("/api/user/orders", srv.addOrder)
		r.Get("/api/user/orders", srv.getOrders)
		r.Get("/api/user/balance", srv.getBalance)
		r.Post("/api/user/balance/withdraw", srv.withdraw)
		r.Get("/api/user/withdrawals", srv.getWithdrawals)
	})

	return router
}
