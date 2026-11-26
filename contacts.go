package main

// ContactSpan is one inclusive frame range where a foot stays planted.
type ContactSpan struct {
	Foot int
	Lo   int
	Hi   int
}

// ContactSpans extracts planted ranges from per-frame contact flags.
func ContactSpans(clip *Clip) []ContactSpan {
	var spans []ContactSpan
	for foot := 0; foot < 2; foot++ {
		start := -1
		for f := 0; f <= len(clip.Frames); f++ {
			active := f < len(clip.Frames) && clip.Frames[f].Contacts[foot]
			if active && start < 0 {
				start = f
			}
			if !active && start >= 0 {
				spans = append(spans, ContactSpan{Foot: foot, Lo: start, Hi: f - 1})
				start = -1
			}
		}
	}
	return spans
}

// ReconstructContacts rebuilds per-frame flags from decoded contact spans.
func ReconstructContacts(n int, spans []ContactSpan) [][2]bool {
	out := make([][2]bool, n)
	for _, s := range spans {
		for f := s.Lo; f <= s.Hi && f < n; f++ {
			out[f][s.Foot] = true
		}
	}
	return out
}
