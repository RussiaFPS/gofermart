package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/RussiaFPS/gofermart/internal/model"
	"github.com/RussiaFPS/gofermart/internal/utils"
	"io"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (h *Handler) userRegstr(rw http.ResponseWriter, r *http.Request) {
	user, err := h.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.service.RgstrUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrLoginExists) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = utils.AddAuthorizationHeader(rw, user, h.cfg.SecretKey); err != nil {
		h.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.log.Info("Пользователь успешно зарегистрирован и аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) userAuth(rw http.ResponseWriter, r *http.Request) {
	user, err := h.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	h.log.WithFields(logrus.Fields{
		"user": user.Login}).Info("Аутентификация пользователя")

	err = h.service.AuthUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrAuthFailed) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = utils.AddAuthorizationHeader(rw, user, h.cfg.SecretKey); err != nil {
		h.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.log.Info("Пользователь успешно аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) addOrder(rw http.ResponseWriter, r *http.Request) {
	h.log.Info("Добавить заказ")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Add order handler| %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	number := string(body)

	if r.Header.Get("Content-Type") != "text/plain" {
		h.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		h.log.Error(model.ErrCastingType)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.service.AddUserOrder(r.Context(), number, login)
	if err != nil {
		if errors.Is(err, model.ErrOrderExistsSameUser) {
			rw.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, model.ErrOrderExistsDiffUser) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, model.ErrNotValidOrderNumber) {
			rw.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusAccepted)
}

func (h *Handler) getOrders(rw http.ResponseWriter, r *http.Request) {
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		h.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
	}

	orders, err := h.service.GetUserOrders(r.Context(), login)
	if err != nil {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(orders)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	if _, err = rw.Write(buf.Bytes()); err != nil {
		h.log.Error("Не удалось написать ответ:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Info("Список заказов успешно возвращен")
}

func (h *Handler) withdraw(rw http.ResponseWriter, r *http.Request) {
	var withdraw model.OrderWithdraw

	h.log.Info("Попытка списания средств")

	if r.Header.Get("Content-Type") != "application/json" {
		h.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&withdraw); err != nil {
		h.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		h.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.service.WriteWithdraw(r.Context(), withdraw, login); err != nil {
		if errors.Is(err, model.ErrNotValidOrderNumber) {
			rw.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, model.ErrInsufficientBalance) {
			rw.WriteHeader(http.StatusPaymentRequired)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Info("Списание произошло")
	rw.WriteHeader(http.StatusOK)
}

func (h *Handler) getWithdrawals(rw http.ResponseWriter, r *http.Request) {
	h.log.Info("Получение информации о выводе средств")
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		h.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	withdrawals, err := h.service.GetWithdrawals(r.Context(), login)
	if err != nil {
		if errors.Is(err, model.ErrNoWithdrawals) {
			rw.WriteHeader(http.StatusNoContent)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(withdrawals)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.WithFields(logrus.Fields{"withdrawals": withdrawals}).Info("Информация о выводе средств получена")
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if _, err = rw.Write(buf.Bytes()); err != nil {
		h.log.Error("Не удалось написать ответ:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) getBalance(rw http.ResponseWriter, r *http.Request) {
	h.log.Info("Получение баланса")

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		h.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := h.service.GetBalance(r.Context(), login)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	buf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(balance)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	if _, err = rw.Write(buf.Bytes()); err != nil {
		h.log.Error("Не удалось написать ответ:", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.log.Info("Баланс пользователя успешно возвращен")
}
