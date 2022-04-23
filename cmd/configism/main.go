package main

import (
	"github.com/amannm/configism/pkg/cmd"
	"os"
)

func main() {
	exitCode := cmd.Execute()
	os.Exit(exitCode)
}
