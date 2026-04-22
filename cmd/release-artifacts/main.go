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
	var (
		repoRoot   = flag.String("repo-root", ".", "repository root")
		version    = flag.String("version", "", "release version")
		commitSHA  = flag.String("commit", "", "git commit sha")
		tag        = flag.String("tag", "", "git tag")
		outputDir  = flag.String("output", "", "artifact output directory")
		dockerTags = flag.String("docker-tags", "", "comma-separated docker tags")
	)
	flag.Parse()

	result, err := releaseartifacts.Generate(releaseartifacts.Options{
		RepoRoot:   mustAbs(*repoRoot),
		Version:    strings.TrimSpace(*version),
		CommitSHA:  strings.TrimSpace(*commitSHA),
		Tag:        strings.TrimSpace(*tag),
		OutputDir:  strings.TrimSpace(*outputDir),
		DockerTags: splitCSV(*dockerTags),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(result.OutputDir)
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return strings.TrimSpace(path)
	}
	return abs
}

func splitCSV(value string) []string {
	parts := strings.Split(strings.TrimSpace(value), ",")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
