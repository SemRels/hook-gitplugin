package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type call struct {
	dir  string
	name string
	args []string
}

type fakeRunner struct {
	calls []call
	errs  map[string]error
}

func (f *fakeRunner) run(_ context.Context, dir string, name string, args ...string) (string, error) {
	copied := append([]string(nil), args...)
	f.calls = append(f.calls, call{dir: dir, name: name, args: copied})
	if err := f.errs[key(name, args...)]; err != nil {
		return "", err
	}
	return "ok", nil
}

func key(name string, args ...string) string {
	out := name
	for _, arg := range args {
		out += " " + arg
	}
	return out
}

func TestNewPluginDefaults(t *testing.T) {
	p := NewPlugin(Config{TagName: "v{version}"})
	if p.cfg.Remote != "origin" || p.cfg.Branch != "main" {
		t.Fatalf("unexpected defaults: %+v", p.cfg)
	}
	if p.Name() != "git" || p.Version() == "" {
		t.Fatalf("unexpected metadata")
	}
}

func TestValidate(t *testing.T) {
	if err := NewPlugin(Config{}).Validate(); err == nil {
		t.Fatal("expected validation error")
	}
	if err := NewPlugin(Config{TagName: "v{version}"}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTagLightweight(t *testing.T) {
	runner := &fakeRunner{}
	p := NewPlugin(Config{TagName: "v{version}"})
	p.runner = runner

	if err := p.CreateTag(context.Background(), "C:\\repo", "1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []call{{dir: "C:\\repo", name: "git", args: []string{"tag", "v1.2.3"}}}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("unexpected calls: %#v", runner.calls)
	}
}

func TestCreateTagAnnotatedAndSigned(t *testing.T) {
	runner := &fakeRunner{}
	p := NewPlugin(Config{TagName: "v{version}", TagMessage: "release {version}", SignTag: true})
	p.runner = runner

	if err := p.CreateTag(context.Background(), "C:\\repo", "1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"tag", "-s", "-a", "-m", "release 1.2.3", "v1.2.3"}
	if !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("unexpected args: %#v", runner.calls[0].args)
	}
}

func TestCreateTagError(t *testing.T) {
	runner := &fakeRunner{errs: map[string]error{key("git", "tag", "v1.2.3"): errors.New("boom")}}
	p := NewPlugin(Config{TagName: "v{version}"})
	p.runner = runner

	if err := p.CreateTag(context.Background(), "C:\\repo", "1.2.3"); err == nil {
		t.Fatal("expected create tag error")
	}
}

func TestCommitFilesNoFiles(t *testing.T) {
	runner := &fakeRunner{}
	p := NewPlugin(Config{TagName: "v{version}"})
	p.runner = runner

	if err := p.CommitFiles(context.Background(), "C:\\repo", "1.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("expected no calls, got %#v", runner.calls)
	}
}

func TestCommitFilesDefaultMessageAndSignoff(t *testing.T) {
	runner := &fakeRunner{}
	p := NewPlugin(Config{TagName: "v{version}", Files: []string{"CHANGELOG.md"}, SignedOffBy: true})
	p.runner = runner

	if err := p.CommitFiles(context.Background(), "C:\\repo", "1.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []call{
		{dir: "C:\\repo", name: "git", args: []string{"add", "CHANGELOG.md"}},
		{dir: "C:\\repo", name: "git", args: []string{"commit", "-m", "chore: release 1.0.0", "--signoff"}},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("unexpected calls: %#v", runner.calls)
	}
}

func TestCommitFilesStageError(t *testing.T) {
	runner := &fakeRunner{errs: map[string]error{key("git", "add", "CHANGELOG.md"): errors.New("boom")}}
	p := NewPlugin(Config{TagName: "v{version}", Files: []string{"CHANGELOG.md"}})
	p.runner = runner

	if err := p.CommitFiles(context.Background(), "C:\\repo", "1.0.0"); err == nil {
		t.Fatal("expected stage error")
	}
}

func TestCommitFilesCommitError(t *testing.T) {
	runner := &fakeRunner{errs: map[string]error{key("git", "commit", "-m", "release 1.0.0"): errors.New("boom")}}
	p := NewPlugin(Config{TagName: "v{version}", Files: []string{"CHANGELOG.md"}, CommitMessage: "release {version}"})
	p.runner = runner

	if err := p.CommitFiles(context.Background(), "C:\\repo", "1.0.0"); err == nil {
		t.Fatal("expected commit error")
	}
}

func TestPushSuccess(t *testing.T) {
	runner := &fakeRunner{}
	p := NewPlugin(Config{TagName: "v{version}", Remote: "upstream", Branch: "release"})
	p.runner = runner

	if err := p.Push(context.Background(), "C:\\repo", "1.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []call{
		{dir: "C:\\repo", name: "git", args: []string{"push", "upstream", "release"}},
		{dir: "C:\\repo", name: "git", args: []string{"push", "upstream", "v1.0.0"}},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("unexpected calls: %#v", runner.calls)
	}
}

func TestPushBranchError(t *testing.T) {
	runner := &fakeRunner{errs: map[string]error{key("git", "push", "origin", "main"): errors.New("boom")}}
	p := NewPlugin(Config{TagName: "v{version}"})
	p.runner = runner

	if err := p.Push(context.Background(), "C:\\repo", "1.0.0"); err == nil {
		t.Fatal("expected push branch error")
	}
}

func TestPushTagError(t *testing.T) {
	runner := &fakeRunner{errs: map[string]error{key("git", "push", "origin", "v1.0.0"): errors.New("boom")}}
	p := NewPlugin(Config{TagName: "v{version}"})
	p.runner = runner

	if err := p.Push(context.Background(), "C:\\repo", "1.0.0"); err == nil {
		t.Fatal("expected push tag error")
	}
}

func TestExpandTagName(t *testing.T) {
	if got := ExpandTagName("v{version}", "1.2.3"); got != "v1.2.3" {
		t.Fatalf("unexpected expanded tag: %s", got)
	}
	if got := ExpandTagName("static", "1.2.3"); got != "static" {
		t.Fatalf("unexpected expanded tag: %s", got)
	}
}
