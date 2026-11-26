package demo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"shadowcache/internal/cache"
	"shadowcache/internal/render"
)

type FileInfo struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	SHA256 string `json:"sha256"`
}

type Scenario struct {
	ID              string      `json:"id"`
	Description     string      `json:"description"`
	Output          string      `json:"output,omitempty"`
	RequestedShadow string      `json:"requested_shadow"`
	ReturnedShadow  string      `json:"returned_shadow"`
	Key             string      `json:"key"`
	KeyDigest       string      `json:"key_digest"`
	Hit             bool        `json:"cache_hit"`
	WrongShader     bool        `json:"wrong_shader"`
	PixelsCompared  bool        `json:"pixels_compared"`
	ReferenceEqual  bool        `json:"reference_equal"`
	Diff            render.Diff `json:"diff_vs_reference"`
	PixelHash       string      `json:"pixel_hash"`
	ReferenceHash   string      `json:"reference_hash"`
}

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type Report struct {
	Schema         string      `json:"schema"`
	ShaderVersion  string      `json:"shader_version"`
	SceneVersion   string      `json:"scene_version"`
	LegacyKey      string      `json:"legacy_key_example"`
	FixedKeyHard   string      `json:"fixed_key_hard"`
	FixedKeySoft   string      `json:"fixed_key_soft"`
	Outputs        Outputs     `json:"outputs"`
	Scenarios      []Scenario  `json:"scenarios"`
	Checks         []Check     `json:"checks"`
	FixedCache     CacheReport `json:"fixed_cache"`
	LegacyCache    CacheReport `json:"legacy_cache"`
	AllChecksPass  bool        `json:"all_checks_pass"`
	Deterministic  bool        `json:"deterministic_output"`
	PolicyNotes    []string    `json:"policy_notes"`
	ValidationNote string      `json:"validation_note"`
}

type Outputs struct {
	Correct  FileInfo `json:"correct"`
	WrongKey FileInfo `json:"wrong_key"`
	Rebuilt  FileInfo `json:"rebuilt"`
	Report   string   `json:"report"`
}

type CacheReport struct {
	Stats  cache.Stats   `json:"stats"`
	Events []cache.Event `json:"events"`
}

func fileInfo(path string, frame *render.Frame) (FileInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{}, err
	}
	sum := sha256.Sum256(data)
	return FileInfo{
		Path:   filepath.ToSlash(path),
		Bytes:  len(data),
		Width:  frame.Width,
		Height: frame.Height,
		SHA256: hex.EncodeToString(sum[:]),
	}, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func scenarioFromResult(id, description, output string, r cache.Result) Scenario {
	key := r.Key.String()
	keyDigest := r.Key.Digest()
	if len(r.Events) > 0 {
		key = r.Events[0].Key
		keyDigest = r.Events[0].KeyDigest
	}
	s := Scenario{
		ID:              id,
		Description:     description,
		Output:          output,
		RequestedShadow: requestedShadow(r),
		ReturnedShadow:  r.Program.Mode.String(),
		Key:             key,
		KeyDigest:       keyDigest,
		Hit:             r.Hit,
		WrongShader:     r.WrongShader,
		PixelsCompared:  r.Diff.TotalPixels > 0,
		ReferenceEqual:  r.Diff.Equal(),
		Diff:            r.Diff,
	}
	if r.Frame != nil {
		s.PixelHash = r.Frame.Hash()
	}
	if r.Reference != nil {
		s.ReferenceHash = r.Reference.Hash()
	}
	return s
}

func requestedShadow(r cache.Result) string {
	if len(r.Events) == 0 {
		return r.Key.ShadowMode.String()
	}
	return r.Events[0].RequestedShadow
}

func check(name string, ok bool, format string, args ...any) Check {
	return Check{Name: name, OK: ok, Detail: fmt.Sprintf(format, args...)}
}
