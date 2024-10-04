package common

func GetMapEntries(m map[string]any, keys []string) []any {
	entries := []any{}
	for _, key := range keys {
		entries = append(entries, m[key])
	}
	return entries
}
