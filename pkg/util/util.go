package util

func IsContained(arr []string, val string) bool {
	for _, str := range arr {
		if str == val {
			return true
		}
	}

	return false
}
