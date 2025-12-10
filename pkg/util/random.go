package util

import (
	"math/rand"
	"time"
)

// GenerateRandomNumber generates a random number between min and max (inclusive)
func GenerateRandomNumber(min, max int) int {
	// Seed with current time to ensure randomness
	rand.Seed(time.Now().UnixNano())
	return min + rand.Intn(max-min+1)
}
