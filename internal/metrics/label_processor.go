package metrics

import (
	"regexp"
	"strings"
)

var (
	namespacePattern = regexp.MustCompile(`namespace="([^"]+)"`)
	podPattern       = regexp.MustCompile(`pod="([^"]+)"`)
)

// LabelProcessor enriches Prometheus metrics with additional labels sourced
// from pod metadata or default value fallbacks.
type LabelProcessor struct{}

// NewLabelProcessor returns a ready-to-use LabelProcessor.
func NewLabelProcessor() *LabelProcessor {
	return &LabelProcessor{}
}

// AddLabelsToMetrics walks the metrics payload and appends the requested
// labels to lines that already contain pod and namespace labels.
func (lp *LabelProcessor) AddLabelsToMetrics(
	metrics,
	addLabels,
	labelDefaults string,
	resolvePodLabels func(namespace, podName string) map[string]string,
) string {
	targetLabels := splitLabels(addLabels)
	if len(targetLabels) == 0 {
		return metrics
	}

	defaultValues := parseLabelDefaults(labelDefaults)

	var builder strings.Builder
	lines := strings.Split(metrics, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			builder.WriteString(line)
			builder.WriteByte('\n')
			continue
		}

		if strings.Contains(line, "{") && strings.Contains(line, "}") {
			line = lp.processMetricLine(line, targetLabels, defaultValues, resolvePodLabels)
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String()
}

func (lp *LabelProcessor) processMetricLine(
	line string,
	targetLabels []string,
	defaultValues map[string]string,
	resolvePodLabels func(namespace, podName string) map[string]string,
) string {
	namespace, podName := extractNamespaceAndPod(line)
	if namespace == "" || podName == "" {
		return line
	}

	podLabels := resolvePodLabels(namespace, podName)
	mutated := line

	for _, label := range targetLabels {
		if strings.Contains(mutated, label+"=") {
			continue
		}

		value := labelValue(label, podLabels, defaultValues)
		if value == "" {
			continue
		}

		pos := strings.LastIndex(mutated, "}")
		if pos == -1 {
			continue
		}

		head := mutated[:pos]
		tail := strings.TrimSpace(mutated[pos+1:])
		mutated = head + `,` + label + `="` + escapeLabelValue(value) + `"}` + tail
	}

	return mutated
}

func extractNamespaceAndPod(line string) (namespace, pod string) {
	if matches := namespacePattern.FindStringSubmatch(line); len(matches) == 2 {
		namespace = matches[1]
	}
	if matches := podPattern.FindStringSubmatch(line); len(matches) == 2 {
		pod = matches[1]
	}
	return
}

func splitLabels(labels string) []string {
	if labels == "" {
		return nil
	}

	parts := strings.Split(labels, ",")
	out := make([]string, 0, len(parts))
	for _, label := range parts {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseLabelDefaults(defaults string) map[string]string {
	defaultMap := make(map[string]string)
	if defaults == "" || defaults == "null" {
		return defaultMap
	}

	pairs := strings.Split(defaults, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		if !strings.Contains(pair, "=") {
			defaultMap["__global__"] = pair
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := ""
		if len(parts) == 2 {
			value = strings.TrimSpace(parts[1])
		}

		if key != "" {
			defaultMap[key] = value
		}
	}

	return defaultMap
}

func labelValue(label string, podLabels map[string]string, defaults map[string]string) string {
	if podLabels != nil {
		if value := strings.TrimSpace(podLabels[label]); value != "" {
			return value
		}
	}

	if value := strings.TrimSpace(defaults[label]); value != "" {
		return value
	}

	return strings.TrimSpace(defaults["__global__"])
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}
