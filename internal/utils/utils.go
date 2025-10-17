package utils

import (
	"github.com/RussiaFPS/gofermart/internal/model"
	"math"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

const (
	asciiZero = 48
)

func CheckLuhnAlg(number string) bool {
	var luhn int64

	p := (len(number)) % 2

	for i, v := range number {
		v = v - asciiZero
		if i%2 == p {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		luhn += int64(v)
	}
	return luhn%10 == 0
}

func Round(x float64, prec int) float64 {
	var rounder float64

	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)

	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}

func AddAuthorizationHeader(rw http.ResponseWriter, user model.User, secret string) error {
	tk := &model.Token{Login: user.Login}
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}
	rw.Header().Add("Authorization", tokenString)
	return nil
}
