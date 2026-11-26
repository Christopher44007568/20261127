package bloom

import (
	"fmt"
	"math"
)

type NodeBase struct {
	KindName string
}

func (n NodeBase) Kind() string { return n.KindName }

func (NodeBase) AcceptsInput(f FormatID) bool {
	_, err := FormatInfoFor(f)
	return err == nil
}

func (NodeBase) SupportsOutput(f FormatID) bool {
	_, err := FormatInfoFor(f)
	return err == nil
}

type SourceNode struct {
	NodeBase
	Generate func(width, height int) (*Image, error)
}

func NewSourceNode(generate func(width, height int) (*Image, error)) *SourceNode {
	return &SourceNode{NodeBase: NodeBase{KindName: "source"}, Generate: generate}
}

func (*SourceNode) InputCount() int { return 0 }

func (n *SourceNode) Evaluate(_ []*Image, _ FormatID) (*Image, error) {
	if n.Generate == nil {
		return nil, fmt.Errorf("source has no generator")
	}
	return n.Generate(0, 0)
}

type ThresholdNode struct {
	NodeBase
	Threshold float64
	Knee      float64
}

func NewThresholdNode(threshold, knee float64) (*ThresholdNode, error) {
	if threshold < 0 || knee < 0 {
		return nil, fmt.Errorf("threshold and knee must be non-negative")
	}
	return &ThresholdNode{
		NodeBase: NodeBase{KindName: "threshold"}, Threshold: threshold, Knee: knee,
	}, nil
}

func (*ThresholdNode) InputCount() int { return 1 }

func (n *ThresholdNode) Evaluate(inputs []*Image, _ FormatID) (*Image, error) {
	src := inputs[0]
	out := src.Clone()
	for i, c := range src.Pixels {
		luma := c.Luma()
		low := n.Threshold - n.Knee
		high := n.Threshold + n.Knee
		var weight float64
		switch {
		case luma >= high:
			weight = 1
		case luma > low && high > low:
			t := (luma - low) / (high - low)
			weight = t * t * (3 - 2*t)
		}
		out.Pixels[i] = c.Mul(weight)
	}
	return out, nil
}

type BlurAxis int

const (
	BlurHorizontal BlurAxis = iota
	BlurVertical
)

type BlurNode struct {
	NodeBase
	Axis   BlurAxis
	Radius int
	Sigma  float64
}

func NewBlurNode(axis BlurAxis, radius int, sigma float64) (*BlurNode, error) {
	if radius < 1 {
		return nil, fmt.Errorf("blur radius must be at least 1")
	}
	if sigma <= 0 {
		return nil, fmt.Errorf("blur sigma must be positive")
	}
	name := "blur.horizontal"
	if axis == BlurVertical {
		name = "blur.vertical"
	}
	return &BlurNode{NodeBase: NodeBase{KindName: name}, Axis: axis, Radius: radius, Sigma: sigma}, nil
}

func (*BlurNode) InputCount() int { return 1 }

func (n *BlurNode) Evaluate(inputs []*Image, _ FormatID) (*Image, error) {
	src := inputs[0]
	out, err := NewImage(src.Width, src.Height)
	if err != nil {
		return nil, err
	}
	weights := make([]float64, n.Radius*2+1)
	var total float64
	for i := range weights {
		d := float64(i - n.Radius)
		w := math.Exp(-(d * d) / (2 * n.Sigma * n.Sigma))
		weights[i] = w
		total += w
	}
	for i := range weights {
		weights[i] /= total
	}
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			var sum RGB
			for k, weight := range weights {
				d := k - n.Radius
				sx, sy := x, y
				if n.Axis == BlurHorizontal {
					sx = clampInt(x+d, 0, src.Width-1)
				} else {
					sy = clampInt(y+d, 0, src.Height-1)
				}
				sum = sum.Add(src.At(sx, sy).Mul(weight))
			}
			out.Set(x, y, sum)
		}
	}
	return out, nil
}

type ScaleNode struct {
	NodeBase
	Factor float64
}

func NewScaleNode(factor float64) *ScaleNode {
	return &ScaleNode{NodeBase: NodeBase{KindName: "scale"}, Factor: factor}
}

func (*ScaleNode) InputCount() int { return 1 }

func (n *ScaleNode) Evaluate(inputs []*Image, _ FormatID) (*Image, error) {
	out := inputs[0].Clone()
	for i := range out.Pixels {
		out.Pixels[i] = out.Pixels[i].Mul(n.Factor)
	}
	return out, nil
}

type CompositeNode struct {
	NodeBase
	BloomStrength float64
}

func NewCompositeNode(bloomStrength float64) *CompositeNode {
	return &CompositeNode{
		NodeBase: NodeBase{KindName: "composite"}, BloomStrength: bloomStrength,
	}
}

func (*CompositeNode) InputCount() int { return 2 }

func (n *CompositeNode) Evaluate(inputs []*Image, _ FormatID) (*Image, error) {
	source, bloom := inputs[0], inputs[1]
	if err := sameDimensions(source, bloom); err != nil {
		return nil, err
	}
	out := source.Clone()
	for i := range out.Pixels {
		out.Pixels[i] = source.Pixels[i].Add(bloom.Pixels[i].Mul(n.BloomStrength))
	}
	return out, nil
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
