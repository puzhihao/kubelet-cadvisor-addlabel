package metrics

import (
	"sort"
	"strconv"
	"strings"
)

const relationMetricName = "kubelet_cadvisor_label_relation"

func buildRelationMetrics(service *Service, labelKeys []string, defaults map[string]string) string {
	if service == nil || len(labelKeys) == 0 {
		return ""
	}

	var builder strings.Builder
	metricsWritten := false

	for _, key := range labelKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		set := make(map[string]struct{})
		for _, value := range service.UniqueLabelValues(key) {
			value = strings.TrimSpace(value)
			if value != "" {
				set[value] = struct{}{}
			}
		}
		if def := relationDefaultValue(key, defaults); def != "" {
			set[def] = struct{}{}
		}

		if len(set) == 0 {
			continue
		}

		values := make([]string, 0, len(set))
		for v := range set {
			values = append(values, v)
		}
		sort.Strings(values)

		if !metricsWritten {
			builder.WriteString("# HELP ")
			builder.WriteString(relationMetricName)
			builder.WriteString(" Hash of unique pod label values requested via ADD_LABELS.\n")
			builder.WriteString("# TYPE ")
			builder.WriteString(relationMetricName)
			builder.WriteString(" gauge\n")
			metricsWritten = true
		}

		for _, value := range values {
			hash := labelRelationHash(value)
			builder.WriteString(relationMetricName)
			builder.WriteString(`{label_key="`)
			builder.WriteString(escapeLabelValue(key))
			builder.WriteString(`",label_value="`)
			builder.WriteString(escapeLabelValue(value))
			builder.WriteString(`"} `)
			builder.WriteString(strconv.FormatUint(hash, 10))
			builder.WriteByte('\n')
		}
	}

	if !metricsWritten {
		return ""
	}

	return builder.String()
}

func relationDefaultValue(label string, defaults map[string]string) string {
	if len(defaults) == 0 {
		return ""
	}

	if value := strings.TrimSpace(defaults[label]); value != "" {
		return value
	}

	return strings.TrimSpace(defaults["__global__"])
}
