package tools

import (
	"context"
	"testing"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

func TestEvaluate_FileConfiguredToolRequiresInstalledBinary(t *testing.T) {
	def := toolFileConfigured(
		"fake",
		"Fake Tool",
		"test",
		"definitely-not-installed-all-cli-test-binary",
		func() (bool, []string, []string) {
			return true, nil, nil
		},
	)

	summary := Evaluate(context.Background(), def, execx.DefaultRunner{})

	if summary.Installed {
		t.Fatalf("expected binary to be missing, got installed summary %#v", summary)
	}
	if summary.ConfiguredState != model.ConfiguredUnknown {
		t.Fatalf("expected configured state to stay unknown when binary is missing, got %#v", summary.ConfiguredState)
	}
	if summary.Configured {
		t.Fatalf("expected configured=false when binary is missing, got %#v", summary)
	}
}

func TestEvaluate_FileConfiguredToolKeepsConfiguredWhenInstalled(t *testing.T) {
	def := toolFileConfigured(
		"fake",
		"Fake Tool",
		"test",
		"go",
		func() (bool, []string, []string) {
			return true, []string{"dup", "dup"}, []string{"err", "err"}
		},
	)
	def.Current = func(_ context.Context, _ execx.Runner, installed bool) (map[string]string, []string, []string) {
		if !installed {
			t.Fatalf("expected installed binary for current callback")
		}
		return map[string]string{"profile": "dev"}, []string{"dup"}, []string{"err"}
	}

	summary := Evaluate(context.Background(), def, execx.DefaultRunner{})

	if !summary.Installed {
		t.Fatalf("expected go binary to be installed, got %#v", summary)
	}
	if summary.ConfiguredState != model.ConfiguredYes || !summary.Configured {
		t.Fatalf("expected configured yes for installed binary, got %#v", summary)
	}
	if len(summary.Warnings) != 1 || summary.Warnings[0] != "dup" {
		t.Fatalf("expected deduped warnings, got %#v", summary.Warnings)
	}
	if len(summary.Errors) != 1 || summary.Errors[0] != "err" {
		t.Fatalf("expected deduped errors, got %#v", summary.Errors)
	}
	if summary.Current["profile"] != "dev" {
		t.Fatalf("expected current context to be preserved, got %#v", summary.Current)
	}
}
