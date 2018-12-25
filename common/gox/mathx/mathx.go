// Package mathx contains maths utilites
package mathx

func MaxInt(xs ...int) int {
	if len(xs) == 0 {
		panic("empty")
	} else if len(xs) == 1 {
		return xs[0]
	}

	var max = xs[0]
	for _, x := range xs[1:] {
		if x > max {
			max = x
		}
	}

	return max
}

func MinInt(xs ...int) int {
	if len(xs) == 0 {
		panic("empty")
	} else if len(xs) == 1 {
		return xs[0]
	}

	var min = xs[0]
	for _, x := range xs[1:] {
		if x < min {
			min = x
		}
	}

	return min
}
