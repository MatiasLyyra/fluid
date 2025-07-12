package simulation

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Vector2 struct {
	X, Y float32
}

func (v Vector2) AsRaylib() rl.Vector2 {
	return rl.Vector2{
		X: v.X,
		Y: v.Y,
	}
}

func (v Vector2) Add(v2 Vector2) Vector2 {
	return Vector2{X: v.X + v2.X, Y: v.Y + v2.Y}
}

// AddValue - Add vector and float value
func (v Vector2) AddValue(add float32) Vector2 {
	return Vector2{X: v.X + add, Y: v.Y + add}
}

// Subtract - Subtract two vectors (v1 - v2)
func (v Vector2) Subtract(v2 Vector2) Vector2 {
	return Vector2{X: v.X - v2.X, Y: v.Y - v2.Y}
}

// SubtractValue - Subtract vector by float value
func (v Vector2) SubtractValue(sub float32) Vector2 {
	return Vector2{X: v.X - sub, Y: v.Y - sub}
}

// Length - Calculate vector length
func (v Vector2) Length() float32 {
	return float32(math.Sqrt(float64((v.X * v.X) + (v.Y * v.Y))))
}

// LengthSqr - Calculate vector square length
func (v Vector2) LengthSqr() float32 {
	return v.X*v.X + v.Y*v.Y
}

// DotProduct - Calculate two vectors dot product
func (v Vector2) DotProduct(v2 Vector2) float32 {
	return v.X*v2.X + v.Y*v2.Y
}

// Distance - Calculate distance between two vectors
func (v Vector2) Distance(v2 Vector2) float32 {
	return float32(math.Sqrt(float64((v.X-v2.X)*(v.X-v2.X) + (v.Y-v2.Y)*(v.Y-v2.Y))))
}

// DistanceSqr - Calculate square distance between two vectors
func (v Vector2) DistanceSqr(v2 Vector2) float32 {
	return (v.X-v2.X)*(v.X-v2.X) + (v.Y-v2.Y)*(v.Y-v2.Y)
}

// Angle - Calculate angle from two vectors in radians
func (v Vector2) Angle(v2 Vector2) float32 {
	result := math.Atan2(float64(v2.Y), float64(v2.X)) - math.Atan2(float64(v.Y), float64(v.X))
	return float32(result)
}

// LineAngle - Calculate angle defined by a two vectors line
// NOTE: Parameters need to be normalized. Current implementation should be aligned with glm::angle
func (v Vector2) LineAngle(end Vector2) float32 {
	return float32(-math.Atan2(float64(end.Y-v.Y), float64(end.X-v.X)))
}

// Scale - Scale vector (multiply by value)
func (v Vector2) Scale(scale float32) Vector2 {
	return Vector2{X: v.X * scale, Y: v.Y * scale}
}

// Multiply - Multiply vector by vector
func (v Vector2) Multiply(v1, v2 Vector2) Vector2 {
	return Vector2{X: v1.X * v2.X, Y: v1.Y * v2.Y}
}

// Negate - Negate vector
func (v Vector2) Negate() Vector2 {
	return Vector2{X: -v.X, Y: -v.Y}
}

// Divide - Divide vector by vector
func (v Vector2) Divide(v2 Vector2) Vector2 {
	return Vector2{X: v.X / v2.X, Y: v.Y / v2.Y}
}

// Normalize - Normalize provided vector
func (v Vector2) Normalize() Vector2 {
	if l := v.Length(); l > 0 {
		return v.Scale(1 / l)
	}
	return v
}

// Transform - Transforms a Vector2 by a given Matrix
func (v Vector2) Transform(mat rl.Matrix) Vector2 {
	var result = Vector2{}

	var x = v.X
	var y = v.Y
	var z float32

	result.X = mat.M0*x + mat.M4*y + mat.M8*z + mat.M12
	result.Y = mat.M1*x + mat.M5*y + mat.M9*z + mat.M13

	return result
}

// Lerp - Calculate linear interpolation between two vectors
func (v Vector2) Lerp(v2 Vector2, amount float32) Vector2 {
	return Vector2{X: v.X + amount*(v2.X-v.X), Y: v.Y + amount*(v2.Y-v.Y)}
}

// Reflect - Calculate reflected vector to normal
func (v Vector2) Reflect(normal Vector2) Vector2 {
	var result = Vector2{}

	dotProduct := v.X*normal.X + v.Y*normal.Y // Dot product

	result.X = v.X - 2.0*normal.X*dotProduct
	result.Y = v.Y - 2.0*normal.Y*dotProduct

	return result
}

// Rotate - Rotate vector by angle
func (v Vector2) Rotate(angle float32) Vector2 {
	var result = Vector2{}

	cosres := float32(math.Cos(float64(angle)))
	sinres := float32(math.Sin(float64(angle)))

	result.X = v.X*cosres - v.Y*sinres
	result.Y = v.X*sinres + v.Y*cosres

	return result
}

// MoveTowards - Move Vector towards target
func (v Vector2) MoveTowards(target Vector2, maxDistance float32) Vector2 {
	var result = Vector2{}

	dx := target.X - v.X
	dy := target.Y - v.Y
	value := dx*dx + dy*dy

	if value == 0 || maxDistance >= 0 && value <= maxDistance*maxDistance {
		return target
	}

	dist := float32(math.Sqrt(float64(value)))

	result.X = v.X + dx/dist*maxDistance
	result.Y = v.Y + dy/dist*maxDistance

	return result
}

// Invert - Invert the given vector
func (v Vector2) Invert() Vector2 {
	return Vector2{X: 1.0 / v.X, Y: 1.0 / v.Y}
}

// Clamp - Clamp the components of the vector between min and max values specified by the given vectors
func (v Vector2) Clamp(min Vector2, max Vector2) Vector2 {
	var result = Vector2{}

	result.X = float32(math.Min(float64(max.X), math.Max(float64(min.X), float64(v.X))))
	result.Y = float32(math.Min(float64(max.Y), math.Max(float64(min.Y), float64(v.Y))))

	return result
}

// ClampValue - Clamp the magnitude of the vector between two min and max values
func (v Vector2) ClampValue(min float32, max float32) Vector2 {
	var result = v

	length := v.X*v.X + v.Y*v.Y
	if length > 0.0 {
		length = float32(math.Sqrt(float64(length)))

		if length < min {
			scale := min / length
			result.X = v.X * scale
			result.Y = v.Y * scale
		} else if length > max {
			scale := max / length
			result.X = v.X * scale
			result.Y = v.Y * scale
		}
	}

	return result
}

// Equals - Check whether two given vectors are almost equal
func (v Vector2) Equals(q Vector2) bool {
	return (math.Abs(float64(v.X-q.X)) <= 0.000001*math.Max(1.0, math.Max(math.Abs(float64(v.X)), math.Abs(float64(q.X)))) &&
		math.Abs(float64(v.Y-q.Y)) <= 0.000001*math.Max(1.0, math.Max(math.Abs(float64(v.Y)), math.Abs(float64(q.Y)))))
}

// CrossProduct - Calculate two vectors cross product
func (v Vector2) CrossProduct(v2 Vector2) float32 {
	return v.X*v2.Y - v.Y*v2.X
}

// Cross - Calculate the cross product of a vector and a value
func (v Vector2) Cross(value float32, vector Vector2) Vector2 {
	return Vector2{X: -value * vector.Y, Y: value * vector.X}
}

// LenSqr - Returns the len square root of a vector
func (v Vector2) LenSqr() float32 {
	return v.X*v.X + v.Y*v.Y
}
