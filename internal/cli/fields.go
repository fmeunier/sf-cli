package cli

import (
	"fmt"
	"strings"
)

func parseFields(raw string, provided bool, allowed []string, command string) ([]string, error) {
	if !provided {
		return nil, nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedSet[field] = struct{}{}
	}

	fields := make([]string, 0, len(allowed))
	seen := make(map[string]struct{}, len(allowed))
	for _, part := range strings.Split(raw, ",") {
		field := strings.TrimSpace(part)
		if field == "" {
			return nil, fmt.Errorf("--fields requires a comma-separated list of field names")
		}
		if _, ok := allowedSet[field]; !ok {
			return nil, fmt.Errorf("unsupported --fields value %q for %s; supported fields: %s", field, command, strings.Join(allowed, ", "))
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		fields = append(fields, field)
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("--fields requires a comma-separated list of field names")
	}

	return fields, nil
}
