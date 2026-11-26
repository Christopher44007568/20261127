package render

// spectralColor is a compact 380-780 nm visible-spectrum approximation used
// only for the decorative wavelength band under chart axes. The physically
// correct image colors come from the CIE integration in package color.
func spectralColor(lambda float64) RGB {
	var r, g, b float64
	switch {
	case lambda < 440:
		r, g, b = -(lambda-440)/60, 0, 1
	case lambda < 490:
		r, g, b = 0, (lambda-440)/50, 1
	case lambda < 510:
		r, g, b = 0, 1, -(lambda-510)/20
	case lambda < 580:
		r, g, b = (lambda-510)/70, 1, 0
	case lambda < 645:
		r, g, b = 1, -(lambda-645)/65, 0
	default:
		r, g, b = 1, 0, 0
	}

	// Fade the near-UV and near-IR edges toward darkness.
	factor := 1.0
	switch {
	case lambda < 420:
		factor = 0.3 + 0.7*(lambda-380)/40
	case lambda > 700:
		factor = 0.3 + 0.7*(780-lambda)/80
	}
	return rgb(255*r*factor, 255*g*factor, 255*b*factor)
}
