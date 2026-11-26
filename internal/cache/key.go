package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"shadowcache/internal/render"
)

// Key is complete and versioned. Every input that can alter pixels is present;
// in particular ShadowMode is part of the key, unlike the legacy key below.
type Key struct {
	Schema        uint32
	ShaderVersion string
	SceneVersion  string
	Width         int
	Height        int
	ShadowMode    render.ShadowMode
}

func (k Key) String() string {
	return fmt.Sprintf("v%d|shader=%s|scene=%s|%dx%d|shadow=%s",
		k.Schema, k.ShaderVersion, k.SceneVersion, k.Width, k.Height, k.ShadowMode)
}

func (k Key) Digest() string {
	sum := sha256.Sum256([]byte(k.String()))
	return hex.EncodeToString(sum[:])
}

// LegacyKey reproduces the original defect: it contains the shader and scene
// versions but omits ShadowMode. A hard-shadow program cached under this key is
// therefore returned for a soft-shadow request.
type LegacyKey struct {
	ShaderVersion string
	SceneVersion  string
	Width         int
	Height        int
}

func (k LegacyKey) String() string {
	return fmt.Sprintf("legacy|shader=%s|scene=%s|%dx%d|<shadow-mode-missing>",
		k.ShaderVersion, k.SceneVersion, k.Width, k.Height)
}

func (k LegacyKey) Digest() string {
	sum := sha256.Sum256([]byte(k.String()))
	return hex.EncodeToString(sum[:])
}
