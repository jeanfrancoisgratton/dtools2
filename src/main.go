package main

import (
	"dtools2/cmd"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// We need to create a configuration directory. This is a per-user config dir
	if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2"), os.ModePerm); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cmd.Execute()
}
