package main

import (
	"github.com/joho/godotenv"
)

type Environment map[string]string

func getEnvironment() Environment {
	env, err := godotenv.Read()
	if err != nil {
		panic(err)
	}
	return env
}
