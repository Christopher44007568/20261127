package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeBytes(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", path, err)
	}
	return nil
}

func outputPath(dir, name string) string { return filepath.Join(dir, name) }

func f6(x float64) string { return fmt.Sprintf("%.6f", x) }
