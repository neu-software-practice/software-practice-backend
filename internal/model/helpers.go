package model

// Float64Ptr returns a pointer to the given float64 value.
// Useful for assigning literal values to *float64 struct fields.
func Float64Ptr(v float64) *float64 { return &v }

// DerefFloat64 safely dereferences a *float64 pointer, returning 0 if nil.
func DerefFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
