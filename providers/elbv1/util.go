package awselbv1

// returns sliceA string values that are not in sliceB.
func differenceStringSlice(sliceA []string, sliceB []string) []string {
	var diff []string

	for _, a := range sliceA {
		found := false
		for _, b := range sliceB {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, a)
		}
	}

	return diff
}

// returns a new slice with all duplicate items removed
func removeDuplicates(in []string) (out []string) {
	m := map[string]bool{}
	for _, v := range in {
		if _, found := m[v]; !found {
			out = append(out, v)
			m[v] = true
		}
	}
	return
}
