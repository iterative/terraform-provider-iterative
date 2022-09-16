package main

import (
	"os"
)

func main() {
	cmd := NewCmd()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
