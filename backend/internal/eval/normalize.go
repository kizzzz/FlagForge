package eval

import "reflect"

// Normalize converts built-in real numeric types to the JSON number type.
// Other values are preserved, including the contents of objects and arrays.
func Normalize(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case uintptr:
		return float64(n)
	case float32:
		return float64(n)
	default:
		return v
	}
}

// DeepEqualNormalized compares both sides after applying Normalize.
func DeepEqualNormalized(a, b any) bool {
	return reflect.DeepEqual(Normalize(a), Normalize(b))
}
