package zwave

import "math"

const DimMin = 0
const DimMax = 254

// Gamma correction of rgb component
func norm(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	} else {
		return math.Pow((v+0.055)/1.055, 2.4)
	}
}

func round(f float64) int {
	if f < -0.5 {
		return int(f - 0.5)
	}
	if f > 0.5 {
		return int(f + 0.5)
	}
	return 0
}

func RGBToColorXYDim(r, g, b float64) (x float64, y float64, dim int) {
	// Gamma correct sRGB -> sRGB'
	r = norm(r)
	g = norm(g)
	b = norm(b)
	// Wide RGB D65 conversion formula
	X := r*0.664511 + g*0.154324 + b*0.162028
	Y := r*0.313881 + g*0.668433 + b*0.047685
	Z := r*0.000088 + g*0.072310 + b*0.986039
	// Convert XYZ -> xy
	x = X / (X + Y + Z)
	y = Y / (X + Y + Z)
	if Y > 1 {
		Y = 1
	}
	dim = int(Y * 255)
	return
}

func bound(f float64) float64 {
	if f <= 0 {
		return 0
	} else if f >= 1 {
		return 1
	} else {
		return f
	}
}

func KelvinToRGB(k int) (r, g, b float64) {
	if k < 1000 {
		k = 1000
	} else if k > 40000 {
		k = 40000
	}
	t := float64(k / 100)
	if t <= 66 {
		r = 1
		g = bound((99.4708025861*math.Log(t) - 161.1195681661) / 255)
	} else {
		r = bound((329.698727446 * math.Pow(t-60, -0.1332047592)) / 255)
		g = bound((288.1221695283 * math.Pow(t-60, -0.0755148492)) / 255)
	}
	if t >= 66 {
		b = 1
	} else if t <= 19 {
		b = 0
	} else {
		b = bound((138.5177312231*math.Log(t-10) - 305.0447927307) / 255)
	}
	return
}

func KelvinToColorXYDim(k int) (x float64, y float64, dim int) {
	return RGBToColorXYDim(KelvinToRGB(k))
}

func PercentageToDim(p int) int {
	dim := round(float64(p) * DimMax / 100)
	if dim < DimMin {
		dim = DimMin
	} else if dim > DimMax {
		dim = DimMax
	}
	return dim
}

func DimToPercentage(dim int) int {
	p := round(float64(dim) * 100 / DimMax)
	if p > 100 {
		p = 100
	} else if p < 0 {
		p = 0
	}
	return p
}
