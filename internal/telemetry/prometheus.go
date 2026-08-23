package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type metricKey struct {
	Command string
	Result  string
}

type metricValue struct {
	Count       uint64
	DurationSum float64
}

var prometheusMu sync.Mutex

func recordPrometheus(path, command, result string, duration time.Duration) error {
	prometheusMu.Lock()
	defer prometheusMu.Unlock()

	metrics, err := readPrometheus(path)
	if err != nil {
		return err
	}
	key := metricKey{Command: command, Result: result}
	value := metrics[key]
	value.Count++
	value.DurationSum += duration.Seconds()
	metrics[key] = value
	return writePrometheus(path, metrics)
}

func readPrometheus(path string) (map[metricKey]metricValue, error) {
	metrics := make(map[metricKey]metricValue)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return metrics, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		name, key, value, ok := parseMetricLine(line)
		if !ok {
			continue
		}
		current := metrics[key]
		switch name {
		case "all_cli_command_total":
			current.Count = uint64(value)
		case "all_cli_command_duration_seconds_sum":
			current.DurationSum = value
		}
		metrics[key] = current
	}
	return metrics, scanner.Err()
}

func parseMetricLine(line string) (string, metricKey, float64, bool) {
	open := strings.IndexByte(line, '{')
	close := strings.Index(line, `"} `)
	if open <= 0 || close <= open {
		return "", metricKey{}, 0, false
	}
	name := line[:open]
	labels := line[open+1 : close+2]
	var command, result string
	if _, err := fmt.Sscanf(labels, `command=%q,result=%q`, &command, &result); err != nil {
		return "", metricKey{}, 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(line[close+3:]), 64)
	if err != nil {
		return "", metricKey{}, 0, false
	}
	return name, metricKey{Command: command, Result: result}, value, true
}

func writePrometheus(path string, metrics map[metricKey]metricValue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	keys := make([]metricKey, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Command != keys[j].Command {
			return keys[i].Command < keys[j].Command
		}
		return keys[i].Result < keys[j].Result
	})

	var output strings.Builder
	output.WriteString("# HELP all_cli_command_total Completed all-cli commands.\n")
	output.WriteString("# TYPE all_cli_command_total counter\n")
	for _, key := range keys {
		value := metrics[key]
		fmt.Fprintf(&output, "all_cli_command_total%s %d\n", prometheusLabels(key), value.Count)
	}
	output.WriteString("# HELP all_cli_command_duration_seconds Command duration in seconds.\n")
	output.WriteString("# TYPE all_cli_command_duration_seconds summary\n")
	for _, key := range keys {
		value := metrics[key]
		fmt.Fprintf(&output, "all_cli_command_duration_seconds_sum%s %.6f\n", prometheusLabels(key), value.DurationSum)
		fmt.Fprintf(&output, "all_cli_command_duration_seconds_count%s %d\n", prometheusLabels(key), value.Count)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), ".all-cli-prom-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.WriteString(output.String()); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func prometheusLabels(key metricKey) string {
	return fmt.Sprintf(
		`{command="%s",result="%s"}`,
		escapePrometheusLabel(key.Command),
		escapePrometheusLabel(key.Result),
	)
}

func escapePrometheusLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
