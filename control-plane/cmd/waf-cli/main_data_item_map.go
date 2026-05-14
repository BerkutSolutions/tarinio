package main

func asMappedItem(value any) (map[string]any, bool) {
	mapped, ok := value.(map[string]any)
	return mapped, ok
}
