package utils

import (
	"math/rand"
	"time"
)

func GetRandomItems(inputItems []string, numItems int) []string {
	randomItems := make(map[string]bool)
	for len(randomItems) < numItems {
		// Get a random index
		i := rand.Intn(len(inputItems))

		// If the item is not already selected, add it to the map
		if _, ok := randomItems[inputItems[i]]; !ok {
			randomItems[inputItems[i]] = true
		}
	}

	output := []string{}
	for item := range randomItems {
		output = append(output, item)
	}

	return output
}

func GetRandomBoolean() bool {
	rand.Seed(time.Now().UnixNano()) // Set a new seed for each random number generation

	randomNumber := rand.Intn(2) // Generates a random number between 0 and 1

	return randomNumber == 1 // Returns true if randomNumber is 1, otherwise false
}
