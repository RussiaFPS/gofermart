package model

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Token struct {
	Login string
	jwt.RegisteredClaims
}

type OrdersResponse struct {
	Number  string    `json:"number"`
	Status  string    `json:"status"`
	Accrual float64   `json:"accrual"`
	Time    time.Time `json:"uploaded_at"`
	Login   string
}

type PointsAppResponse struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type OrderWithdraw struct {
	Number   string    `json:"order"`
	Withdraw float64   `json:"sum"`
	Time     time.Time `json:"processed_at"`
}

type Balance struct {
	Balance   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

var (
	ErrWrongRequest        = errors.New("wrong request")
	ErrNotAuthorized       = errors.New("user not authorized")
	ErrLoginExists         = errors.New("login already exists")
	ErrAuthFailed          = errors.New("authentification failed")
	ErrOrderExistsSameUser = errors.New("order number has downloaded by current user")
	ErrOrderExistsDiffUser = errors.New("order number has downloaded by other user")
	ErrNotValidOrderNumber = errors.New("order number is not valid")
	ErrInsufficientBalance = errors.New("insufficient funds")
	ErrNoWithdrawals       = errors.New("no withdrawals")
	ErrCastingType         = errors.New("casting types error")
)

type keyLogin string

const (
	KeyLogin keyLogin = "login"
)
