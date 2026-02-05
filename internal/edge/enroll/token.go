package enroll

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type TokenSource struct {
	CLIValue string
	EnvName  string
	FilePath string
}

func ResolveToken(src TokenSource) (string, error) {
	if strings.TrimSpace(src.CLIValue) != "" {
		return strings.TrimSpace(src.CLIValue), nil
	}
	if src.EnvName != "" {
		if env := strings.TrimSpace(os.Getenv(src.EnvName)); env != "" {
			return env, nil
		}
	}
	if strings.TrimSpace(src.FilePath) != "" {
		b, err := os.ReadFile(src.FilePath)
		if err != nil {
			return "", fmt.Errorf("read token file: %w", err)
		}
		if token := strings.TrimSpace(string(b)); token != "" {
			return token, nil
		}
	}
	return "", errors.New("enrollment token is required (cli/env/file)")
}
