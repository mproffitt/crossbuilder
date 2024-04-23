package utils

// BoolPtr converts a bool value to a pointer.
func BoolPtr(b bool) *bool {
	return &b
}

// StringValue returns the string value if p is not nil or an empty string.
func StringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func StrPtr(s string) *string {
	return &s
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func Float64Ptr(f float64) *float64 {
	return &f
}
