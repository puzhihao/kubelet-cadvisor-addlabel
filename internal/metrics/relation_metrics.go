package metrics

import (
	"strconv"
	"strings"
)

const relationMetricName = "kubelet_cadvisor_label_relation"

func buildRelationMetrics(service *Service, labelKeys []string) string {
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

		values := service.UniqueLabelValues(key)
		if len(values) == 0 {
			continue
		}

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
