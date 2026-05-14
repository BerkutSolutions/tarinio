package main

import "strings"

func asMap(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return item
}

func asList(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := asMappedItem(item); ok {
			out = append(out, mapped)
		}
	}
	return out
}

func asStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text := strings.TrimSpace(stringify(item))
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}
