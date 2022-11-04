package main

import (
	"github.com/vippsas/sqlcode/cli/cmd"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
