package biped

import (
	"math"
	"sort"
)

type Weight struct {
	ID    string  `json:"id"`
	Value float64 `json:"value"`
}

type WeightSample struct {
	TargetSpeed float64  `json:"target_speed"`
	Speed       float64  `json:"speed"`
	Weights     []Weight `json:"weights"`
	Target      []Weight `json:"target_weights"`
	Sum         float64  `json:"sum"`
}

type BlendSpace struct {
	Clips []Clip `json:"clips"`
}

func NewDefaultBlendSpace() *BlendSpace {
	clips := []Clip{
		IdleClip(),
		WalkClip(),
		RunClip(),
	}
	return &BlendSpace{Clips: clips}
}

func (b *BlendSpace) Clip(id string) (Clip, bool) {
	for _, clip := range b.Clips {
		if clip.ID == id {
			return clip, true
		}
	}
	return Clip{}, false
}

func (b *BlendSpace) sortedClips() []Clip {
	clips := append([]Clip(nil), b.Clips...)
	sort.SliceStable(clips, func(i, j int) bool {
		return clips[i].Speed < clips[j].Speed
	})
	return clips
}

func (b *BlendSpace) TargetWeights(speed float64) []Weight {
	clips := b.sortedClips()
	weights := make([]Weight, len(clips))
	for i, clip := range clips {
		weights[i] = Weight{ID: clip.ID}
	}
	if len(clips) == 0 {
		return nil
	}

	speed = math.Max(0, speed)
	if speed <= clips[0].Speed {
		weights[0].Value = 1
		return NormalizeWeights(weights)
	}
	last := len(clips) - 1
	if speed >= clips[last].Speed {
		weights[last].Value = 1
		return NormalizeWeights(weights)
	}

	for i := 0; i < last; i++ {
		a, c := clips[i], clips[i+1]
		if speed < a.Speed || speed > c.Speed || math.Abs(c.Speed-a.Speed) < 1e-12 {
			continue
		}
		t := (speed - a.Speed) / (c.Speed - a.Speed)
		weights[i].Value = 1 - t
		weights[i+1].Value = t
		break
	}
	return NormalizeWeights(weights)
}

func NormalizeWeights(weights []Weight) []Weight {
	var sum float64
	for i := range weights {
		if math.IsNaN(weights[i].Value) || weights[i].Value < 0 {
			weights[i].Value = 0
		}
		sum += weights[i].Value
	}
	if sum <= 1e-12 {
		if len(weights) > 0 {
			weights[0].Value = 1
			sum = 1
		} else {
			return weights
		}
	}
	for i := range weights {
		weights[i].Value /= sum
	}
	return weights
}

func WeightValue(weights []Weight, id string) float64 {
	for _, weight := range weights {
		if weight.ID == id {
			return weight.Value
		}
	}
	return 0
}

func WeightSum(weights []Weight) float64 {
	var sum float64
	for _, weight := range weights {
		sum += weight.Value
	}
	return sum
}

func BlendFloat(weights []Weight, value func(id string) float64) float64 {
	var result float64
	for _, weight := range weights {
		result += weight.Value * value(weight.ID)
	}
	return result
}
