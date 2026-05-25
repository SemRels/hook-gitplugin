package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	plugin "github.com/SemRels/hook-gitplugin/internal/plugin"
)

type gitPlugin interface {
	Validate() error
	CommitFiles(context.Context, string, string) error
	CreateTag(context.Context, string, string) error
	Push(context.Context, string, string) error
}

var (
	newPlugin = func(cfg plugin.Config) gitPlugin { return plugin.NewPlugin(cfg) }
	getwd     = os.Getwd
)

func run(ctx context.Context, getenv func(string) string, stderr io.Writer) int {
	tagName := firstNonEmpty(getenv("SEMREL_PLUGIN_TAG_NAME"), getenv("SEMREL_TAG_NAME"), getenv("SEMREL_VERSION"), getenv("SEMREL_NEXT_VERSION"))
	versionSource := firstNonEmpty(getenv("SEMREL_VERSION"), getenv("SEMREL_TAG_NAME"), getenv("SEMREL_NEXT_VERSION"), tagName)
	if tagName == "" || versionSource == "" {
		fmt.Fprintln(stderr, "hook-gitplugin: SEMREL_PLUGIN_TAG_NAME, SEMREL_TAG_NAME, SEMREL_VERSION, or SEMREL_NEXT_VERSION is required")
		return 1
	}

	cfg := plugin.Config{
		TagName:       tagName,
		TagMessage:    getenv("SEMREL_PLUGIN_TAG_MESSAGE"),
		CommitMessage: getenv("SEMREL_PLUGIN_COMMIT_MESSAGE"),
		Remote:        getenv("SEMREL_PLUGIN_REMOTE"),
		Branch:        firstNonEmpty(getenv("SEMREL_PLUGIN_BRANCH"), getenv("SEMREL_BRANCH")),
		Files:         splitFiles(getenv("SEMREL_PLUGIN_FILES")),
		SignTag:       parseBool(getenv("SEMREL_PLUGIN_SIGN_TAG")),
		SignedOffBy:   parseBool(getenv("SEMREL_PLUGIN_SIGNED_OFF_BY")),
	}

	if getenv("SEMREL_DRY_RUN") == "true" {
		return 0
	}

	p := newPlugin(cfg)
	if err := p.Validate(); err != nil {
		fmt.Fprintln(stderr, "hook-gitplugin:", err)
		return 1
	}

	dir, err := getwd()
	if err != nil {
		fmt.Fprintln(stderr, "hook-gitplugin:", err)
		return 1
	}

	version := normalizeVersion(versionSource, getenv("SEMREL_TAG_PREFIX"))
	if err := p.CommitFiles(ctx, dir, version); err != nil {
		fmt.Fprintln(stderr, "hook-gitplugin:", err)
		return 1
	}
	if err := p.CreateTag(ctx, dir, version); err != nil {
		fmt.Fprintln(stderr, "hook-gitplugin:", err)
		return 1
	}
	if err := p.Push(ctx, dir, version); err != nil {
		fmt.Fprintln(stderr, "hook-gitplugin:", err)
		return 1
	}
	return 0
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	os.Exit(run(ctx, os.Getenv, os.Stderr))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func splitFiles(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			files = append(files, part)
		}
	}
	return files
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizeVersion(version, prefix string) string {
	if prefix != "" {
		return strings.TrimPrefix(version, prefix)
	}
	return strings.TrimPrefix(version, "v")
}
