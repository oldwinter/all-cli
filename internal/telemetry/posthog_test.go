package telemetry

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestLoadOrCreateInstallationIDLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics", "installation-id")

	id, err := loadOrCreateInstallationID(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) {
		t.Fatalf("id = %q, want 32 hex chars", id)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("installation ID file mode = %o, want 0600", info.Mode().Perm())
	}

	again, err := loadOrCreateInstallationID(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again != id {
		t.Fatalf("expected persisted id %q, got %q", id, again)
	}
}

func TestLoadOrCreateInstallationIDReadsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation-id")
	if err := os.WriteFile(path, []byte("  abc123  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := loadOrCreateInstallationID(path)
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" {
		t.Fatalf("expected trimmed existing id, got %q", id)
	}
}

func TestLoadOrCreateInstallationIDRegeneratesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation-id")
	if err := os.WriteFile(path, []byte(" \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := loadOrCreateInstallationID(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 32 {
		t.Fatalf("expected regenerated id, got %q", id)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != id+"\n" {
		t.Fatalf("file not rewritten with new id: %q", raw)
	}
}

func TestNewPosthogSinkDefaultsAndEndpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation-id")

	sink, err := newPosthogSink(Config{PostHogKey: "k", Release: "1.0", InstallationIDPath: path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sink.endpoint != "https://us.i.posthog.com/capture/" {
		t.Fatalf("endpoint = %q", sink.endpoint)
	}
	if len(sink.installationID) != 32 {
		t.Fatalf("installationID = %q", sink.installationID)
	}
	if sink.apiKey != "k" || sink.release != "1.0" {
		t.Fatalf("config fields not wired: %+v", sink)
	}
}

func TestNewPosthogSinkCustomHostTrimsSlash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation-id")
	sink, err := newPosthogSink(Config{PostHogHost: "https://eu.i.posthog.com/", InstallationIDPath: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sink.endpoint != "https://eu.i.posthog.com/capture/" {
		t.Fatalf("endpoint = %q", sink.endpoint)
	}
}

func TestNewPosthogSinkRejectsBadHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation-id")
	for _, host := range []string{"notaurl", "ftp://", "  "} {
		// "  " trims to empty and falls back to the default host, so only
		// scheme-less inputs should fail.
		_, err := newPosthogSink(Config{PostHogHost: host, InstallationIDPath: path}, nil)
		if host == "  " {
			if err != nil {
				t.Fatalf("blank host should use default, got %v", err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("expected error for host %q", host)
		}
	}
}

func TestLoadOrCreateInstallationIDErrorsOnUnreadablePath(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadOrCreateInstallationID(dir); err == nil {
		t.Fatal("expected error reading a directory as the ID file")
	}
}

func TestLoadOrCreateInstallationIDErrorsOnUncreatableDir(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateInstallationID(filepath.Join(blocker, "installation-id")); err == nil {
		t.Fatal("expected error creating ID under a file")
	}
}
