package main

import (
	"os"
	"terraform-provider-iterative/iterative/utils"
)

func main() {
	defer utils.WaitForAnalyticsAndHandlePanics()
	cmd := NewCmd()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
