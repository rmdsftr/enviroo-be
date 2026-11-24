package utils

import "math"

func Round4(x float64) float64 {
	return math.Round(x*1e4) / 1e4
}
