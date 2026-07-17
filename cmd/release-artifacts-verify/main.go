package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"waf/internal/releaseartifacts"
)

func main() {
	output := flag.String("output", "", "release artifacts directory")
	trustedPublicKey := flag.String("trusted-public-key", "", "protected trusted Ed25519 public key PEM path")
	flag.Parse()

	dir := strings.TrimSpace(*output)
	if dir == "" {
		fmt.Fprintln(os.Stderr, "output directory is required")
		os.Exit(1)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	if err := releaseartifacts.Verify(absDir, strings.TrimSpace(*trustedPublicKey)); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("release artifacts verification passed")
}
