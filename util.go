package main

import (
	"os"
	"strconv"
)

func getEnvAsInt(name string) int {
	str := os.Getenv(name)
	i, _ := strconv.Atoi(str)
	return i
}
