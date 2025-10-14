package main

import (
	"fmt"
	"os"

	"github.com/toshsan/scaffold/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: scaffold <template.yaml|URL|github.com/...> [args...]")
		os.Exit(1)
	}

	templateFile := os.Args[1]
	args := os.Args[2:]

	if err := scaffold.Run(templateFile, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
