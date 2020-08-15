package util

import (
	"math/rand"
	"time"
)

// GetDailyWord ...
func GetDailyWord() string {
	words := []string{"Pandas are the best.",
		"It’s panda day.",
		"Jk, everyday is panda day.",
		"It’s still panda day.",
		"You know what day it is.",
		"What day is it?",
		"Panda. Panda. Panda.",
		"Panda.",
		"It’s time to panda.",
		"Just panda.",
		"Got panda?",
		"Need more pandas.",
	}

	rand.Seed(time.Now().Unix())
	return words[rand.Intn(len(words))]
}
