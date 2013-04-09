package utils

import (
	"bytes"
	"math/rand"
	"time"
)

const alpha = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func RandomString(l int) string {
	var result bytes.Buffer
	var temp string
	for i := 0; i < l; {
		c := randChar()
		if c != temp {
			temp = c
			result.WriteString(temp)
			i++
		}
	}
	return result.String()
}

func randChar() string {
	rand.Seed(time.Now().UTC().UnixNano())
	return string(alpha[rand.Intn(len(alpha)-1)])
}
