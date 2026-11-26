package cache

// Event is an auditable record of one cache decision. Timing fields are
// diagnostic only; PixelsMatch is the authoritative correctness signal.
type Event struct {
	Seq             int    `json:"seq"`
	Kind            string `json:"kind"`
	Key             string `json:"key"`
	KeyDigest       string `json:"key_digest"`
	RequestedShadow string `json:"requested_shadow"`
	ReturnedShadow  string `json:"returned_shadow,omitempty"`
	SchemaVersion   uint32 `json:"schema_version"`
	ProgramID       string `json:"program_id,omitempty"`
	CompileMicros   int64  `json:"compile_us,omitempty"`
	LookupMicros    int64  `json:"lookup_us,omitempty"`
	RenderMicros    int64  `json:"render_us,omitempty"`
	VerifyMicros    int64  `json:"verify_us,omitempty"`
	PixelsCompared  bool   `json:"pixels_compared"`
	PixelsMatch     bool   `json:"pixels_match"`
	PixelHash       string `json:"pixel_hash,omitempty"`
	ReferenceHash   string `json:"reference_hash,omitempty"`
	DifferentPixels int    `json:"different_pixels,omitempty"`
	Detail          string `json:"detail,omitempty"`
}

type Stats struct {
	Hits          int    `json:"hits"`
	Misses        int    `json:"misses"`
	Invalidations int    `json:"invalidations"`
	Entries       int    `json:"entries"`
	SchemaVersion uint32 `json:"schema_version"`
}
