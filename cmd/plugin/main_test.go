package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	plugin "github.com/SemRels/hook-gitplugin/internal/plugin"
)

type fakePlugin struct {
	validateErr error
	commitErr   error
	tagErr      error
	pushErr     error
	calls       []string
	version     string
	dir         string
}

func (f *fakePlugin) Validate() error { return f.validateErr }
func (f *fakePlugin) CommitFiles(_ context.Context, dir, version string) error {
	f.calls = append(f.calls, "commit")
	f.dir, f.version = dir, version
	return f.commitErr
}
func (f *fakePlugin) CreateTag(_ context.Context, dir, version string) error {
	f.calls = append(f.calls, "tag")
	f.dir, f.version = dir, version
	return f.tagErr
}
func (f *fakePlugin) Push(_ context.Context, dir, version string) error {
	f.calls = append(f.calls, "push")
	f.dir, f.version = dir, version
	return f.pushErr
}

func env(kv map[string]string) func(string) string {
	return func(key string) string { return kv[key] }
}

func TestRunSuccess(t *testing.T) {

	fake := &fakePlugin{}
	oldPlugin, oldGetwd := newPlugin, getwd
	newPlugin = func(cfg plugin.Config) gitPlugin {
		if cfg.TagName != "v1.2.3" || cfg.Branch != "main" || len(cfg.Files) != 2 {
			t.Fatalf("unexpected config: %+v", cfg)
		}
		return fake
	}
	getwd = func() (string, error) { return "C:\\repo", nil }
	defer func() { newPlugin, getwd = oldPlugin, oldGetwd }()

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{
		"SEMREL_TAG_NAME":             "v1.2.3",
		"SEMREL_BRANCH":               "main",
		"SEMREL_PLUGIN_FILES":         "CHANGELOG.md, README.md",
		"SEMREL_PLUGIN_SIGN_TAG":      "true",
		"SEMREL_PLUGIN_SIGNED_OFF_BY": "true",
	}), &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("unexpected result: code=%d stderr=%q", code, stderr.String())
	}
	if got := len(fake.calls); got != 3 {
		t.Fatalf("expected 3 calls, got %v", fake.calls)
	}
	if fake.version != "1.2.3" || fake.dir != "C:\\repo" {
		t.Fatalf("unexpected run args: dir=%s version=%s", fake.dir, fake.version)
	}
}

func TestRunDryRun(t *testing.T) {

	called := false
	oldPlugin := newPlugin
	newPlugin = func(plugin.Config) gitPlugin {
		called = true
		return &fakePlugin{}
	}
	defer func() { newPlugin = oldPlugin }()

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{
		"SEMREL_TAG_NAME": "v1.2.3",
		"SEMREL_DRY_RUN":  "true",
	}), &stderr)
	if code != 0 || called {
		t.Fatalf("unexpected result: code=%d called=%v", code, called)
	}
}

func TestRunValidationError(t *testing.T) {

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{}), &stderr)
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("unexpected result: code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunPluginError(t *testing.T) {

	oldPlugin, oldGetwd := newPlugin, getwd
	newPlugin = func(plugin.Config) gitPlugin { return &fakePlugin{commitErr: errors.New("boom")} }
	getwd = func() (string, error) { return "C:\\repo", nil }
	defer func() { newPlugin, getwd = oldPlugin, oldGetwd }()

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{"SEMREL_TAG_NAME": "v1.2.3"}), &stderr)
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("unexpected result: code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunGetwdError(t *testing.T) {
	oldPlugin, oldGetwd := newPlugin, getwd
	newPlugin = func(plugin.Config) gitPlugin { return &fakePlugin{} }
	getwd = func() (string, error) { return "", errors.New("boom") }
	defer func() { newPlugin, getwd = oldPlugin, oldGetwd }()

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{"SEMREL_TAG_NAME": "v1.2.3"}), &stderr)
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("unexpected result: code=%d stderr=%q", code, stderr.String())
	}
}

func TestHelpers(t *testing.T) {
	if got := firstNonEmpty("", "v1.2.3", "v1.2.4"); got != "v1.2.3" {
		t.Fatalf("unexpected firstNonEmpty result: %s", got)
	}
	files := splitFiles("CHANGELOG.md, README.md\n docs.md")
	if len(files) != 3 {
		t.Fatalf("unexpected files: %v", files)
	}
	if !parseBool("yes") || parseBool("no") {
		t.Fatalf("unexpected parseBool results")
	}
	if got := normalizeVersion("v1.2.3", "v"); got != "1.2.3" {
		t.Fatalf("unexpected normalized version: %s", got)
	}
}

func TestRunValidateError(t *testing.T) {
	oldPlugin := newPlugin
	newPlugin = func(plugin.Config) gitPlugin { return &fakePlugin{validateErr: errors.New("invalid")} }
	defer func() { newPlugin = oldPlugin }()

	var stderr bytes.Buffer
	code := run(context.Background(), env(map[string]string{"SEMREL_TAG_NAME": "v1.2.3"}), &stderr)
	if code != 1 || stderr.Len() == 0 {
		t.Fatalf("unexpected result: code=%d stderr=%q", code, stderr.String())
	}
}
