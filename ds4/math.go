package ds4

import "math"

// GyroRoll returns the gyroscope roll value in degrees between -180 and 180.
// Left roll is negative, right is positive.
func GyroRoll(xi, yi, zi int16) float64 {
	x, y, z := gyroVec(xi, yi, zi)
	wr := math.Copysign(math.Sqrt(y*y+z*z), y)
	return math.Atan2(-x, wr) * 180 / math.Pi
}

// GyroPitch returns the gyroscope roll value in degrees between -180 and 180.
// Pitch down is positive, up is negative.
func GyroPitch(xi, yi, zi int16) float64 {
	x, y, z := gyroVec(xi, yi, zi)
	wp := math.Copysign(math.Sqrt(x*x+y*y), y)
	return math.Atan2(z, wp) * 180 / math.Pi
}

// GyroRollPitch returns the gyroscope pitch and roll values in degrees.
// Roll is between -180..180 and pitch is between -90..90 degrees.
//
// The roll angle becomes unstable when pitch is near ±90° degrees.
func GyroRollPitch(xi, yi, zi int16) (r, p float64) {

	// http://www.nxp.com/files/sensors/doc/app_note/AN3461.pdf
	//
	// In paper:
	//   x: up, y: right, z: back
	//   roll:  φ
	//   pitch: θ
	//
	//   25. tan φ_xyz = y/z
	//   26. tan θ_xyz = -x/√(y²+z²)
	//   28. tan φ_yxz = y/√(x²+z²)
	//   29. tan θ_yxz = -x/z
	//   37. tan θ_xyz = -x/√(y²+z²)  (same as 26.)
	//   38. tan φ_xyz = y/(sign(z)√(z²+μx²)

	// mu is a constant to stabilise the roll value when both x and y
	// is near, that is when pitch approaches ±90°.
	const mu = 0.01

	x, y, z := gyroVec(xi, yi, zi)
	wr := math.Copysign(math.Sqrt(y*y+mu*z*z), y)
	wp := math.Sqrt(x*x + y*y)
	r = math.Atan2(x, wr) * 180 / math.Pi
	p = math.Atan2(z, wp) * 180 / math.Pi
	return r, p
}

// gyroVec returns a normalized based on the gyroscope sensor input
//
// x axis points left
// y axis points down
// z axis points forward
//
// The neutral position is (0 1 0).
func gyroVec(xi, yi, zi int16) (x, y, z float64) {
	x, y, z = float64(xi), float64(yi), float64(zi)
	mag := math.Sqrt(x*x + y*y + z*z)
	x /= mag
	y /= mag
	z /= mag
	return
}
