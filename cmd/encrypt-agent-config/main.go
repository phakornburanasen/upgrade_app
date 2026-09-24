package main

import (
	"flag"
	"fmt"
	"os"

	"upgrade_app/internal/agent"
)

func main() {
	inPath := flag.String("in", "config/agent.json", "plain agent config path")
	outPath := flag.String("out", "config/agent.enc", "encrypted agent config path")
	flag.Parse()

	plain, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	encrypted, err := agent.EncryptConfigBytes(plain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encrypt config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, encrypted, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Printf("Encrypted %s -> %s\n", *inPath, *outPath)
}
