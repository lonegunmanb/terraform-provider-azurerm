package main

func removeFromSlice(s []string, toRemove ...string) []string {
	toRemoveMap := map[string]bool{}
	for _, item := range toRemove {
		toRemoveMap[item] = true
	}

	res := make([]string, 0, len(s))
	for _, item := range s {
		if toRemoveMap[item] {
			continue
		}
		res = append(res, item)
	}

	return res
}
