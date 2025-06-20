package utils

// Helper functions to check string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Helper functions to remove string from a slice of strings.
func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func UniqueValues(dict map[string]string) []string {
	seen := make(map[string]struct{})
	var uniqueValues []string
	for _, value := range dict {
		if _, found := seen[value]; !found {
			seen[value] = struct{}{}
			uniqueValues = append(uniqueValues, value)
		}
	}
	return uniqueValues
}

func UniqueArrayValues(dict map[string][]string) []string {
	seen := make(map[string]struct{})
	var uniqueValues []string
	for _, array := range dict {
		for _, value := range array {
			if _, found := seen[value]; !found {
				seen[value] = struct{}{}
				uniqueValues = append(uniqueValues, value)
			}
		}
	}
	return uniqueValues
}
