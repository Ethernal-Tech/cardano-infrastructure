package common

// SplitString splits large string into slice of substrings
func SplitString(s string, mxlen int) (res []string) {
	for i := 0; i < len(s); i += mxlen {
		res = append(res, s[i:min(i+mxlen, len(s))])
	}

	return res
}
