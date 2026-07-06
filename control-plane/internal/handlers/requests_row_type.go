package handlers

import "strings"

const (
	requestRowTypeRequest  = "request"
	requestRowTypeSecurity = "security"
)

func normalizeRequestRows(items []map[string]any) []map[string]any {
	if len(items) == 0 {
		return items
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, normalizeRequestRow(item))
	}
	return out
}

func normalizeRequestRow(item map[string]any) map[string]any {
	if item == nil {
		return map[string]any{"row_type": requestRowTypeRequest}
	}
	copyRow := make(map[string]any, len(item)+3)
	for key, value := range item {
		copyRow[key] = value
	}
	copyRow["row_type"] = inferLegacyRequestRowType(copyRow)
	copyRow["legacy_row_type_support"] = true
	normalizeRequestSecurityFields(copyRow)
	return copyRow
}

func inferLegacyRequestRowType(item map[string]any) string {
	explicit := normalizeRequestToken(asStringValue(item["row_type"]))
	if explicit == requestRowTypeSecurity {
		return requestRowTypeSecurity
	}
	if explicit == requestRowTypeRequest {
		return requestRowTypeRequest
	}
	explicit = normalizeRequestToken(asStringValue(item["rowType"]))
	if explicit == requestRowTypeSecurity {
		return requestRowTypeSecurity
	}
	if explicit == requestRowTypeRequest {
		return requestRowTypeRequest
	}
	stream := normalizeRequestToken(asStringValue(item["stream"]))
	typeValue := normalizeRequestToken(asStringValue(item["type"]))
	source := normalizeRequestToken(asStringValue(item["source_component"]))
	if strings.HasPrefix(stream, "security") || strings.HasPrefix(typeValue, "security") || strings.Contains(typeValue, "modsecurity") || strings.HasPrefix(source, "security") {
		return requestRowTypeSecurity
	}
	return requestRowTypeRequest
}

func normalizeRequestToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	lastUnderscore := false
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func asStringValue(value any) string {
	text, _ := value.(string)
	return text
}

func normalizeRequestSecurityFields(row map[string]any) {
	if row == nil {
		return
	}
	details, _ := row["details"].(map[string]any)
	eventType := normalizeRequestToken(asStringValue(details["event_type"]))
	if eventType != "" {
		row["event_type"] = eventType
	}
	securityReason := firstNonEmptyRequestString(eventType, asStringValue(row["summary"]), asStringValue(row["type"]), asStringValue(details["intel_label"]), asStringValue(details["feed"]), asStringValue(details["path"]))
	if securityReason != "" {
		row["security_reason"] = securityReason
	}
}

func firstNonEmptyRequestString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
