package helper

func ContainsString(slice []string, elem string) bool {
	for _, a := range slice {
		if a == elem {
			return true
		}
	}
	return false
}
