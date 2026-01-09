package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/simukka/sqlcode/cli/cmd"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
