package analyzer

import "maps"

func copyMap(origMap map[string]struct{}) map[string]struct{} {
	newMap := make(map[string]struct{}, len(origMap))
	maps.Copy(newMap, origMap)

	return newMap
}

func mapHas(m map[string]struct{}, val string) bool {
	_, ok := m[val]

	return ok
}
