package ds4

import "math"

// GyroRoll returns the gyroscope roll value in degrees.
//
// -90: max left roll
// 0: neutral
// 90: max right roll
//
// Note that when rolling more than 90° the absolute value decreases.
func GyroRoll(xi, yi, zi int16) float64 {
	x, y, z := gyroVec(xi, yi, zi)
	wr := math.Sqrt(y*y + z*z)
	return -math.Atan2(x, wr) * 180 / math.Pi
}

// GyroPitch returns the gyroscope pitch value in degrees.
//
// -90: max forward pitch
// 0: neutral
// 90: max backward pitch
//
// Note that when pitching more than 90° the absolute value decreases.
func GyroPitch(xi, yi, zi int16) float64 {
	x, y, z := gyroVec(xi, yi, zi)
	wp := math.Sqrt(x*x + y*y)
	return -math.Atan2(z, wp) * 180 / math.Pi
}

const sqrt3 = 1.73205080756887729352744634150587236694280525381038062805580697 // http://oeis.org/A002194

// GyroRollPitch returns the gyroscope pitch and roll values in degrees.
// Using this function should be preferred over calling GyroRoll and GyroPitch
// separately.
//
// With large roll and pitch input a gimbal lock occurs,
// when it is impossible to calculate accurate roll and pitch values.
// The ok flag is set to false in this case.
// Roll and pitch can be controlled reliably and independently
// only if the absolute value of both are under 45°,
// but in practice an even smaller limit should be used.
func GyroRollPitch(xi, yi, zi int16) (r, p float64, ok bool) {
	x, y, z := gyroVec(xi, yi, zi)
	wr := math.Sqrt(y*y + z*z)
	wp := math.Sqrt(x*x + y*y)
	r = -math.Atan2(x, wr) * 180 / math.Pi
	p = -math.Atan2(z, wp) * 180 / math.Pi

	// TODO(tajti): ask mathematician about limit check
	return r, p, math.Abs(y) > (1 - sqrt3/2)
}

// gyroVec returns a normalized based on the gyroscope sensor input
func gyroVec(xi, yi, zi int16) (x, y, z float64) {
	x, y, z = float64(xi), float64(yi), float64(zi)
	mag := math.Sqrt(x*x + y*y + z*z)
	x /= mag
	y /= mag
	z /= mag
	return
}
