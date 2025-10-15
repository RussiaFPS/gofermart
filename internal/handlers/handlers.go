package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RussiaFPS/gofermart/internal/model"
	"github.com/RussiaFPS/gofermart/internal/utils"
	"io"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (s server) userRegstr(rw http.ResponseWriter, r *http.Request) {
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.service.RgstrUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrLoginExists) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = utils.AddAuthoriztionHeader(rw, user); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.log.Info("Пользователь успешно зарегистрирован и аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (s server) userAuth(rw http.ResponseWriter, r *http.Request) {
	user, err := s.service.ParseUserCredentials(r)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	s.log.WithFields(logrus.Fields{
		"user": user.Login}).Info("Аутентификация пользователя")

	err = s.service.AuthUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, model.ErrAuthFailed) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = utils.AddAuthoriztionHeader(rw, user); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.log.Info("Пользователь успешно аутентифицирован")
	rw.WriteHeader(http.StatusOK)
}

func (s server) addOrder(rw http.ResponseWriter, r *http.Request) {

	s.log.Info("Добавить заказ")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Add order handler| %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	number := string(body)

	if r.Header.Get("Content-Type") != "text/plain" {
		s.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.service.AddUserOrder(r.Context(), number, login)
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

func (s server) getOrders(rw http.ResponseWriter, r *http.Request) {
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
	}

	orders, err := s.service.GetUserOrders(r.Context(), login)
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

	s.log.Info("Список заказов успешно возвращен")
	fmt.Fprint(rw, buf)
}

func (s server) withdraw(rw http.ResponseWriter, r *http.Request) {
	var withdraw model.OrderWithdraw

	s.log.Info("Попытка списания средств")

	if r.Header.Get("Content-Type") != "application/json" {
		s.log.Error("Неверный Content-Type")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&withdraw); err != nil {
		s.log.Error(err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.service.WriteWithdraw(r.Context(), withdraw, login); err != nil {
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
	s.log.Info("Списание произошло")
	rw.WriteHeader(http.StatusOK)
}

func (s server) getWithdrawals(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Получение информации о выводе средств")
	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	withdrawals, err := s.service.GetWithdrawals(r.Context(), login)
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
	s.log.WithFields(logrus.Fields{"withdrawals": withdrawals}).Info("Информация о выводе средств получена")
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, buf)
}

func (s server) getBalance(rw http.ResponseWriter, r *http.Request) {
	s.log.Info("Получение баланса")

	login, ok := r.Context().Value(model.KeyLogin).(string)
	if !ok {
		s.log.Error(model.ErrCastingType.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := s.service.GetBalance(r.Context(), login)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
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

	s.log.Info("Баланс пользователя успешно возвращен")
	fmt.Fprint(rw, buf)

}
