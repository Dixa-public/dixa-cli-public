package main

import (
	"fmt"
	"os"

	"github.com/Dixa-public/dixa-cli-public/internal/cli"
)

func main() {
	env, err := cli.NewDefaultEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize CLI: %v\n", err)
		os.Exit(1)
	}

	cmd := cli.NewRootCmd(env)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(env.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
