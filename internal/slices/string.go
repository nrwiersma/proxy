package slices

// StringContains determines if the string slice contains the string.
func StringContains(s string, ss []string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}

	return false
}
