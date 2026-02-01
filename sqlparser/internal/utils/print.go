package utils

import (
	"fmt"
	"os"
)

var _, enable_debug = os.LookupEnv("SQLCODE_DEBUG")

func DPrint(format string, a ...any) {
	if !enable_debug {
		return
	}
	fmt.Fprintf(os.Stdout, "\033[0;31mDEBUG:\033[0m")
	fmt.Fprintf(os.Stdout, format, a...)
}
