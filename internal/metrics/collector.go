package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

const (
	defaultRequestTimeout = 8 * time.Second
	cadvisorEndpoint      = "https://%s:10250/metrics/cadvisor"
)

// Collector fetches metrics from kubelet cadvisor endpoints and decorates the
// payload with configured labels.
type Collector struct {
	service   *Service
	tokenFile string
	client    *http.Client
	processor *LabelProcessor
}

// NewCollector returns a Collector backed by the provided service cache.
func NewCollector(service *Service, tokenFile string) *Collector {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // kubernetes clusters typically use a self-signed cert
		},
	}

	return &Collector{
		service:   service,
		tokenFile: tokenFile,
		client: &http.Client{
			Timeout:   defaultRequestTimeout,
			Transport: tr,
		},
		processor: NewLabelProcessor(),
	}
}

// Collect retrieves the metrics payload from every known node and applies the
// configured label enrichment rules.
func (c *Collector) Collect(ctx context.Context, addLabels, labelDefaults string) (string, error) {
	nodeIPs := c.service.NodeIPs()
	if len(nodeIPs) == 0 {
		return "", fmt.Errorf("no node IPs available for scraping")
	}

	token, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return "", fmt.Errorf("read service account token: %w", err)
	}

	tokenString := strings.TrimSpace(string(token))
	if tokenString == "" {
		return "", fmt.Errorf("service account token is empty")
	}

	startTime := time.Now()
	klog.InfoS("starting cadvisor scrape", "nodes", len(nodeIPs))

	results := make(map[string]string, len(nodeIPs))
	failures := make(map[string]error)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, ip := range nodeIPs {
		ip := ip
		wg.Add(1)

		go func() {
			defer wg.Done()

			data, err := c.fetchNode(ctx, ip, tokenString)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				failures[ip] = err
				return
			}

			results[ip] = data
		}()
	}

	wg.Wait()

	for ip, err := range failures {
		klog.ErrorS(err, "cadvisor scrape failed", "node", ip)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("cadvisor scrape failed for all %d nodes", len(nodeIPs))
	}

	payload := combineMetrics(results)
	klog.InfoS(
		"cadvisor scrape completed",
		"successes", len(results),
		"failures", len(failures),
		"bytes",
		len(payload),
		"duration",
		time.Since(startTime),
	)

	if addLabels == "" {
		return payload, nil
	}

	klog.InfoS("enriching metrics with labels", "labels", addLabels, "defaults", labelDefaults)
	enriched := c.processor.AddLabelsToMetrics(payload, addLabels, labelDefaults, c.service.PodLabels)
	klog.InfoS("metrics enrichment completed", "originalBytes", len(payload), "enrichedBytes", len(enriched))

	return enriched, nil
}

func (c *Collector) fetchNode(ctx context.Context, ip, token string) (string, error) {
	url := fmt.Sprintf(cadvisorEndpoint, ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	klog.V(4).InfoS("fetched cadvisor metrics", "node", ip, "bytes", len(body))
	return string(body), nil
}

func combineMetrics(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}

	ips := make([]string, 0, len(data))
	for ip := range data {
		ips = append(ips, ip)
	}
	sort.Strings(ips)

	var b strings.Builder
	for _, ip := range ips {
		b.WriteString("# -------- Node: ")
		b.WriteString(ip)
		b.WriteString(" --------\n")
		b.WriteString(data[ip])
		if !strings.HasSuffix(data[ip], "\n") {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
