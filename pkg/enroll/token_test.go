package enroll

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTokenPrecedence(t *testing.T) {
	t.Setenv("NKUDO_TEST_TOKEN", "env-token")
	dir := t.TempDir()
	fp := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(fp, []byte("file-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tok, err := ResolveToken(TokenSource{CLIValue: "cli-token", EnvName: "NKUDO_TEST_TOKEN", FilePath: fp})
	if err != nil {
		t.Fatal(err)
	}
	if tok != "cli-token" {
		t.Fatalf("expected cli-token, got %q", tok)
	}

	tok, err = ResolveToken(TokenSource{EnvName: "NKUDO_TEST_TOKEN", FilePath: fp})
	if err != nil {
		t.Fatal(err)
	}
	if tok != "env-token" {
		t.Fatalf("expected env-token, got %q", tok)
	}

	os.Unsetenv("NKUDO_TEST_TOKEN")
	tok, err = ResolveToken(TokenSource{EnvName: "NKUDO_TEST_TOKEN", FilePath: fp})
	if err != nil {
		t.Fatal(err)
	}
	if tok != "file-token" {
		t.Fatalf("expected file-token, got %q", tok)
	}
}
