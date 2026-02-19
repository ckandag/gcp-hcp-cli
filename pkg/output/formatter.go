// Package output provides formatting utilities for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// Format represents an output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// ParseFormat parses a string into a Format, defaulting to text.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	default:
		return FormatText
	}
}

// PrintJSON writes data as indented JSON to the writer.
func PrintJSON(w io.Writer, data interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintResult formats and prints an execution result based on the output format.
func PrintResult(w io.Writer, format Format, data interface{}) error {
	switch format {
	case FormatJSON:
		return PrintJSON(w, data)
	default:
		return PrintJSON(w, data)
	}
}

// Table provides a simple table writer for text output.
type Table struct {
	w       *tabwriter.Writer
	headers []string
}

// NewTable creates a new table with the given headers.
func NewTable(w io.Writer, headers ...string) *Table {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	t := &Table{w: tw, headers: headers}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	return t
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) {
	fmt.Fprintln(t.w, strings.Join(values, "\t"))
}

// Flush writes the table output.
func (t *Table) Flush() error {
	return t.w.Flush()
}

// PrintResourceTable formats Kubernetes-style resource data as a table.
func PrintResourceTable(w io.Writer, data map[string]interface{}, resourceType string) error {
	items, ok := data["items"].([]interface{})
	if !ok {
		if resource, rOk := data["resource"].(map[string]interface{}); rOk {
			items = []interface{}{resource}
		} else {
			return PrintJSON(w, data)
		}
	}

	if len(items) == 0 {
		fmt.Fprintf(w, "No %s found.\n", resourceType)
		return nil
	}

	switch resourceType {
	case "pods":
		return printPodsTable(w, items)
	case "deployments":
		return printDeploymentsTable(w, items)
	case "hostedclusters":
		return printHostedClustersTable(w, items)
	case "services", "svc":
		return printServicesTable(w, items)
	case "namespaces", "ns":
		return printNamespacesTable(w, items)
	case "nodes":
		return printNodesTable(w, items)
	case "events", "ev":
		return printEventsTable(w, items)
	case "configmaps", "cm":
		return printConfigMapsTable(w, items)
	default:
		return printGenericTable(w, items, resourceType)
	}
}

func printPodsTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		status := AsMap(m["status"])

		readyCount, totalCount := podReadyCounts(status)
		podStatus := podEffectiveStatus(status)
		restarts := podRestartCount(status)

		t.AddRow(
			GetString(meta, "namespace"),
			GetString(meta, "name"),
			fmt.Sprintf("%d/%d", readyCount, totalCount),
			podStatus,
			fmt.Sprintf("%d", restarts),
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printDeploymentsTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		spec := AsMap(m["spec"])
		status := AsMap(m["status"])

		desired := getInt(spec, "replicas")
		ready := getInt(status, "readyReplicas")
		updated := getInt(status, "updatedReplicas")
		available := getInt(status, "availableReplicas")

		t.AddRow(
			GetString(meta, "namespace"),
			GetString(meta, "name"),
			fmt.Sprintf("%d/%d", ready, desired),
			fmt.Sprintf("%d", updated),
			fmt.Sprintf("%d", available),
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printHostedClustersTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAMESPACE", "NAME", "VERSION", "PROGRESS", "AVAILABLE", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		spec := AsMap(m["spec"])
		status := AsMap(m["status"])

		release := AsMap(spec["release"])
		version := GetString(release, "image")
		if version == "" {
			version = "<none>"
		} else if len(version) > 40 {
			version = version[:40] + "..."
		}

		progress := GetString(status, "progress")
		available := conditionStatus(status, "Available")

		t.AddRow(
			GetString(meta, "namespace"),
			GetString(meta, "name"),
			version,
			progress,
			available,
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printServicesTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		spec := AsMap(m["spec"])

		t.AddRow(
			GetString(meta, "namespace"),
			GetString(meta, "name"),
			GetString(spec, "type"),
			GetString(spec, "clusterIP"),
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printConfigMapsTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAMESPACE", "NAME", "DATA", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		data := AsMap(m["data"])

		t.AddRow(
			GetString(meta, "namespace"),
			GetString(meta, "name"),
			fmt.Sprintf("%d", len(data)),
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printNamespacesTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAME", "STATUS", "AGE")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		status := AsMap(m["status"])
		t.AddRow(
			GetString(meta, "name"),
			GetString(status, "phase"),
			age(GetString(meta, "creationTimestamp")),
		)
	}
	return t.Flush()
}

func printNodesTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "NAME", "STATUS", "ROLES", "AGE", "VERSION")
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		status := AsMap(m["status"])
		nodeInfo := AsMap(status["nodeInfo"])

		labels := AsMap(meta["labels"])
		roles := nodeRoles(labels)
		ready := conditionStatus(status, "Ready")
		readyStr := "NotReady"
		if ready == "True" {
			readyStr = "Ready"
		}

		t.AddRow(
			GetString(meta, "name"),
			readyStr,
			roles,
			age(GetString(meta, "creationTimestamp")),
			GetString(nodeInfo, "kubeletVersion"),
		)
	}
	return t.Flush()
}

func printEventsTable(w io.Writer, items []interface{}) error {
	t := NewTable(w, "LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE")
	for _, item := range items {
		m := AsMap(item)
		involvedObject := AsMap(m["involvedObject"])
		objRef := fmt.Sprintf("%s/%s", GetString(involvedObject, "kind"), GetString(involvedObject, "name"))

		lastTimestamp := GetString(m, "lastTimestamp")
		if lastTimestamp == "" {
			lastTimestamp = GetString(m, "eventTime")
		}

		t.AddRow(
			age(lastTimestamp),
			GetString(m, "type"),
			GetString(m, "reason"),
			objRef,
			GetString(m, "message"),
		)
	}
	return t.Flush()
}

func printGenericTable(w io.Writer, items []interface{}, resourceType string) error {
	clusterScoped := isClusterScoped(items)
	if clusterScoped {
		t := NewTable(w, "NAME", "AGE")
		for _, item := range items {
			m := AsMap(item)
			meta := AsMap(m["metadata"])
			t.AddRow(
				GetString(meta, "name"),
				age(GetString(meta, "creationTimestamp")),
			)
		}
		_ = t.Flush()
	} else {
		t := NewTable(w, "NAMESPACE", "NAME", "AGE")
		for _, item := range items {
			m := AsMap(item)
			meta := AsMap(m["metadata"])
			t.AddRow(
				GetString(meta, "namespace"),
				GetString(meta, "name"),
				age(GetString(meta, "creationTimestamp")),
			)
		}
		_ = t.Flush()
	}
	fmt.Fprintf(w, "\n%d %s found.\n", len(items), resourceType)
	return nil
}

func isClusterScoped(items []interface{}) bool {
	for _, item := range items {
		m := AsMap(item)
		meta := AsMap(m["metadata"])
		if GetString(meta, "namespace") != "" {
			return false
		}
	}
	return true
}

func nodeRoles(labels map[string]interface{}) string {
	var roles []string
	for key := range labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

// AsMap safely converts an interface to a string map.
func AsMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// GetString retrieves a string value from a map by key.
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func podEffectiveStatus(status map[string]interface{}) string {
	phase := GetString(status, "phase")

	containers, ok := status["containerStatuses"].([]interface{})
	if !ok || len(containers) == 0 {
		initContainers, iOk := status["initContainerStatuses"].([]interface{})
		if iOk {
			for i, ic := range initContainers {
				icm := AsMap(ic)
				state := AsMap(icm["state"])
				if waiting := AsMap(state["waiting"]); len(waiting) > 0 {
					reason := GetString(waiting, "reason")
					if reason != "" {
						return fmt.Sprintf("Init:%s", reason)
					}
					return fmt.Sprintf("Init:%d/%d", i, len(initContainers))
				}
				if terminated := AsMap(state["terminated"]); len(terminated) > 0 {
					if code := getInt(terminated, "exitCode"); code != 0 {
						return "Init:Error"
					}
				}
			}
		}
		return phase
	}

	for _, c := range containers {
		cm := AsMap(c)
		state := AsMap(cm["state"])

		if waiting := AsMap(state["waiting"]); len(waiting) > 0 {
			if reason := GetString(waiting, "reason"); reason != "" {
				return reason
			}
		}
		if terminated := AsMap(state["terminated"]); len(terminated) > 0 {
			if reason := GetString(terminated, "reason"); reason != "" {
				return reason
			}
		}
	}
	return phase
}

func podReadyCounts(status map[string]interface{}) (ready, total int) {
	containers, ok := status["containerStatuses"].([]interface{})
	if !ok {
		return 0, 0
	}
	total = len(containers)
	for _, c := range containers {
		cm := AsMap(c)
		if r, ok := cm["ready"].(bool); ok && r {
			ready++
		}
	}
	return
}

func podRestartCount(status map[string]interface{}) int {
	containers, ok := status["containerStatuses"].([]interface{})
	if !ok {
		return 0
	}
	total := 0
	for _, c := range containers {
		cm := AsMap(c)
		total += getInt(cm, "restartCount")
	}
	return total
}

func conditionStatus(status map[string]interface{}, condType string) string {
	conditions, ok := status["conditions"].([]interface{})
	if !ok {
		return "Unknown"
	}
	for _, c := range conditions {
		cm := AsMap(c)
		if GetString(cm, "type") == condType {
			return GetString(cm, "status")
		}
	}
	return "Unknown"
}

// Age formats a Kubernetes timestamp as a human-readable duration.
func Age(timestamp string) string {
	return age(timestamp)
}

func age(timestamp string) string {
	if timestamp == "" {
		return "<unknown>"
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return formatDuration(time.Since(t))
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// PrintAnalysis renders AI analysis output for a pod in a human-readable format.
func PrintAnalysis(w io.Writer, data map[string]interface{}, namespace string) error {
	name := GetString(data, "name")
	analysis := AsMap(data["analysis"])

	phase := GetString(analysis, "pod_phase")
	if phase == "" {
		phase = "Unknown"
	}
	eventsCount := getInt(analysis, "events_count")
	logLines := getInt(analysis, "log_lines_analyzed")

	fmt.Fprintln(w)
	fmt.Fprintln(w, "POD ANALYSIS")
	fmt.Fprintln(w, "============")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Pod:       %s\n", name)
	fmt.Fprintf(w, "  Namespace: %s\n", namespace)
	fmt.Fprintf(w, "  Phase:     %s\n", phase)
	fmt.Fprintf(w, "  Events:    %d\n", eventsCount)
	fmt.Fprintf(w, "  Logs:      %d lines analyzed\n", logLines)
	fmt.Fprintln(w)

	aiAnalysis := GetString(analysis, "ai_analysis")
	aiError := GetString(analysis, "error")

	if aiError != "" {
		fmt.Fprintln(w, "AI ANALYSIS")
		fmt.Fprintln(w, "===========")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", aiError)
		fmt.Fprintln(w)
		return nil
	}

	if aiAnalysis == "" || aiAnalysis == "<nil>" {
		fmt.Fprintln(w, "AI ANALYSIS")
		fmt.Fprintln(w, "===========")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  No analysis available.")
		fmt.Fprintln(w)
		return nil
	}

	fmt.Fprintln(w, "AI ANALYSIS")
	fmt.Fprintln(w, "===========")
	fmt.Fprintln(w)

	if rendered := renderStructuredAnalysis(w, aiAnalysis); rendered {
		return nil
	}

	fmt.Fprintln(w, aiAnalysis)
	fmt.Fprintln(w)
	return nil
}

// renderStructuredAnalysis attempts to parse the AI response as structured JSON
// and render it in a human-readable format. Returns true if it succeeded.
func renderStructuredAnalysis(w io.Writer, raw string) bool {
	cleaned := stripCodeFence(raw)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return false
	}

	if _, ok := parsed["summary"]; !ok {
		return false
	}

	severity := stringVal(parsed, "severity")
	if severity != "" {
		fmt.Fprintf(w, "  Severity:  %s\n\n", severity)
	}

	if summary := stringVal(parsed, "summary"); summary != "" {
		printSection(w, "Summary", summary)
	}

	if errors := listVal(parsed, "errors_detected"); len(errors) > 0 {
		printListSection(w, "Errors Detected", errors)
	} else if errStr := stringVal(parsed, "errors_detected"); errStr != "" {
		printSection(w, "Errors Detected", errStr)
	}

	if rca := stringVal(parsed, "root_cause"); rca != "" {
		printSection(w, "Root Cause Analysis", rca)
	}

	if actions := listVal(parsed, "recommended_actions"); len(actions) > 0 {
		printNumberedSection(w, "Recommended Actions", actions)
	} else if actStr := stringVal(parsed, "recommended_actions"); actStr != "" {
		printSection(w, "Recommended Actions", actStr)
	}

	fmt.Fprintln(w)
	return true
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}

func stringVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func listVal(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func printSection(w io.Writer, title, body string) {
	fmt.Fprintf(w, "  %s\n", title)
	for _, line := range wrapText(body, 76) {
		fmt.Fprintf(w, "    %s\n", line)
	}
	fmt.Fprintln(w)
}

func printListSection(w io.Writer, title string, items []string) {
	fmt.Fprintf(w, "  %s\n", title)
	for _, item := range items {
		lines := wrapText(item, 72)
		fmt.Fprintf(w, "    • %s\n", lines[0])
		for _, cont := range lines[1:] {
			fmt.Fprintf(w, "      %s\n", cont)
		}
	}
	fmt.Fprintln(w)
}

func printNumberedSection(w io.Writer, title string, items []string) {
	fmt.Fprintf(w, "  %s\n", title)
	for i, item := range items {
		lines := wrapText(item, 72)
		fmt.Fprintf(w, "    %d. %s\n", i+1, lines[0])
		for _, cont := range lines[1:] {
			fmt.Fprintf(w, "       %s\n", cont)
		}
	}
	fmt.Fprintln(w)
}

func wrapText(s string, width int) []string {
	if len(s) <= width {
		return []string{s}
	}
	var lines []string
	for len(s) > width {
		cut := width
		for cut > 0 && s[cut] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = width
		}
		lines = append(lines, s[:cut])
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		lines = append(lines, s)
	}
	return lines
}

// SortItems sorts a list of Kubernetes items by namespace then name.
func SortItems(items []interface{}) {
	sort.Slice(items, func(i, j int) bool {
		mi := AsMap(AsMap(items[i])["metadata"])
		mj := AsMap(AsMap(items[j])["metadata"])
		nsI := GetString(mi, "namespace")
		nsJ := GetString(mj, "namespace")
		if nsI != nsJ {
			return nsI < nsJ
		}
		return GetString(mi, "name") < GetString(mj, "name")
	})
}
