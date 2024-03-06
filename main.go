package main

import (
	"os"

	"github.com/dsh2dsh/check_wg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
