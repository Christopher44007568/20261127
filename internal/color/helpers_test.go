package color

import "thinfilm/internal/optconst"

func d65Illuminant(lambda float64) float64 { return optconst.D65(lambda) }

func cmf(lambda float64) (x, y, z float64) { return optconst.CMF(lambda) }
