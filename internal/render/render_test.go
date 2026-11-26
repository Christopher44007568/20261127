package render

import "testing"

func TestShadowModesChangeEdgePixels(t *testing.T) {
	hard, _, err := RenderDirect(DefaultSettings(ShadowHard))
	if err != nil {
		t.Fatal(err)
	}
	soft, _, err := RenderDirect(DefaultSettings(ShadowSoft))
	if err != nil {
		t.Fatal(err)
	}
	diff := Compare(hard, soft)
	if diff.Equal() {
		t.Fatal("hard and soft shadow modes produced identical pixels")
	}
	if diff.DifferentPixels < 1000 {
		t.Fatalf("expected a substantial penumbra difference, got %s", diff)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	a, _, err := RenderDirect(DefaultSettings(ShadowSoft))
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := RenderDirect(DefaultSettings(ShadowSoft))
	if err != nil {
		t.Fatal(err)
	}
	if !Compare(a, b).Equal() {
		t.Fatalf("uncached reference is not deterministic: %s", Compare(a, b))
	}
}

func TestProgramModeNotRequestModeDrivesPixels(t *testing.T) {
	hardProgram := CompileShader(ShadowHard)
	requestedSoft := DefaultSettings(ShadowSoft)
	requestedHard := DefaultSettings(ShadowHard)

	fromSoftRequest, err := Render(requestedSoft, hardProgram)
	if err != nil {
		t.Fatal(err)
	}
	fromHardRequest, err := Render(requestedHard, hardProgram)
	if err != nil {
		t.Fatal(err)
	}
	if !Compare(fromSoftRequest, fromHardRequest).Equal() {
		t.Fatalf("a stale hard program should render hard pixels even for a soft request: %s",
			Compare(fromSoftRequest, fromHardRequest))
	}
}
