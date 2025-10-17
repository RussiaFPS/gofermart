package handlers

import (
	"compress/gzip"
	"context"
	"github.com/RussiaFPS/gofermart/internal/model"
	"io"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

func (h *Handler) gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		b, err := io.ReadAll(gz)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(b)))

		next.ServeHTTP(w, r)

	})
}

func (h *Handler) checkUserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.log.Info("Проверка аутентификации пользователя")
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			h.log.Error("Токен пуст")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tk := &model.Token{}
		token, err := jwt.ParseWithClaims(tokenHeader, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.cfg.SecretKey), nil
		})

		if err != nil {
			h.log.Error(err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			h.log.Error("Token not valid")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), model.KeyLogin, tk.Login)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
