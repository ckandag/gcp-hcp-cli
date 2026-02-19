package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"30 seconds", 30 * time.Second, "30s"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"2 hours", 2 * time.Hour, "2h"},
		{"3 days", 72 * time.Hour, "3d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.dur); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}
}

func TestConditionStatus(t *testing.T) {
	status := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{"type": "Ready", "status": "True"},
			map[string]interface{}{"type": "Available", "status": "False"},
		},
	}
	if got := conditionStatus(status, "Ready"); got != "True" {
		t.Errorf("expected 'True', got %q", got)
	}
	if got := conditionStatus(status, "Available"); got != "False" {
		t.Errorf("expected 'False', got %q", got)
	}
	if got := conditionStatus(status, "Missing"); got != "Unknown" {
		t.Errorf("expected 'Unknown' for missing condition, got %q", got)
	}
	if got := conditionStatus(map[string]interface{}{}, "Ready"); got != "Unknown" {
		t.Errorf("expected 'Unknown' for no conditions, got %q", got)
	}
}

func TestPodReadyCounts(t *testing.T) {
	tests := []struct {
		name      string
		status    map[string]interface{}
		wantReady int
		wantTotal int
	}{
		{
			name: "When all containers are ready it should report full count",
			status: map[string]interface{}{
				"containerStatuses": []interface{}{
					map[string]interface{}{"ready": true},
					map[string]interface{}{"ready": true},
				},
			},
			wantReady: 2, wantTotal: 2,
		},
		{
			name: "When one container is not ready it should report partial count",
			status: map[string]interface{}{
				"containerStatuses": []interface{}{
					map[string]interface{}{"ready": true},
					map[string]interface{}{"ready": false},
				},
			},
			wantReady: 1, wantTotal: 2,
		},
		{
			name:      "When no container statuses exist it should return zeros",
			status:    map[string]interface{}{},
			wantReady: 0, wantTotal: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, total := podReadyCounts(tt.status)
			if ready != tt.wantReady || total != tt.wantTotal {
				t.Errorf("got %d/%d, want %d/%d", ready, total, tt.wantReady, tt.wantTotal)
			}
		})
	}
}

func TestPodEffectiveStatus(t *testing.T) {
	tests := []struct {
		name   string
		status map[string]interface{}
		want   string
	}{
		{
			name:   "When pod is running it should show Running",
			status: map[string]interface{}{"phase": "Running", "containerStatuses": []interface{}{map[string]interface{}{"state": map[string]interface{}{"running": map[string]interface{}{}}}}},
			want:   "Running",
		},
		{
			name: "When container is in CrashLoopBackOff it should show that reason",
			status: map[string]interface{}{
				"phase": "Running",
				"containerStatuses": []interface{}{
					map[string]interface{}{"state": map[string]interface{}{"waiting": map[string]interface{}{"reason": "CrashLoopBackOff"}}},
				},
			},
			want: "CrashLoopBackOff",
		},
		{
			name: "When init container is waiting it should show Init prefix",
			status: map[string]interface{}{
				"phase": "Pending",
				"initContainerStatuses": []interface{}{
					map[string]interface{}{"state": map[string]interface{}{"waiting": map[string]interface{}{"reason": "ImagePullBackOff"}}},
				},
			},
			want: "Init:ImagePullBackOff",
		},
		{
			name:   "When no containers exist it should fall back to phase",
			status: map[string]interface{}{"phase": "Pending"},
			want:   "Pending",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := podEffectiveStatus(tt.status); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNodeRoles(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]interface{}
		want   string
	}{
		{
			name:   "When labels contain worker and control-plane it should return sorted roles",
			labels: map[string]interface{}{"node-role.kubernetes.io/worker": "", "node-role.kubernetes.io/control-plane": ""},
			want:   "control-plane,worker",
		},
		{
			name:   "When no role labels exist it should return none",
			labels: map[string]interface{}{"app": "test"},
			want:   "<none>",
		},
		{
			name:   "When labels are empty it should return none",
			labels: map[string]interface{}{},
			want:   "<none>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeRoles(tt.labels); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintResourceTable_EmptyItems(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResourceTable(&buf, map[string]interface{}{"items": []interface{}{}}, "pods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No pods found") {
		t.Errorf("expected 'No pods found', got %q", buf.String())
	}
}

func TestPrintResourceTable_SingleResource(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]interface{}{
		"resource": map[string]interface{}{
			"metadata": map[string]interface{}{"name": "my-svc", "namespace": "default", "creationTimestamp": "2025-01-01T00:00:00Z"},
			"spec":     map[string]interface{}{"type": "ClusterIP", "clusterIP": "10.0.0.1"},
		},
	}
	if err := PrintResourceTable(&buf, data, "services"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "my-svc") || !strings.Contains(out, "ClusterIP") {
		t.Errorf("expected service details in output, got:\n%s", out)
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", `{"key":"val"}`, `{"key":"val"}`},
		{"json fence", "```json\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
		{"bare fence", "```\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripCodeFence(tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	short := "short text"
	if lines := wrapText(short, 80); len(lines) != 1 || lines[0] != short {
		t.Errorf("short text should not wrap, got %v", lines)
	}

	long := "This is a much longer sentence that should definitely be wrapped at some reasonable point in the output"
	lines := wrapText(long, 40)
	if len(lines) < 2 {
		t.Errorf("expected multiple lines for long text, got %d", len(lines))
	}
	for _, line := range lines {
		if len(line) > 40 {
			t.Errorf("line exceeds width: %q (%d chars)", line, len(line))
		}
	}
}

func TestSortItems(t *testing.T) {
	items := []interface{}{
		map[string]interface{}{"metadata": map[string]interface{}{"namespace": "b-ns", "name": "pod-1"}},
		map[string]interface{}{"metadata": map[string]interface{}{"namespace": "a-ns", "name": "pod-2"}},
		map[string]interface{}{"metadata": map[string]interface{}{"namespace": "a-ns", "name": "pod-1"}},
	}
	SortItems(items)

	first := AsMap(AsMap(items[0])["metadata"])
	if GetString(first, "namespace") != "a-ns" || GetString(first, "name") != "pod-1" {
		t.Errorf("first item should be a-ns/pod-1, got %s/%s", GetString(first, "namespace"), GetString(first, "name"))
	}
	last := AsMap(AsMap(items[2])["metadata"])
	if GetString(last, "namespace") != "b-ns" {
		t.Errorf("last item should be in b-ns, got %s", GetString(last, "namespace"))
	}
}

func TestPrintAnalysis_WithStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]interface{}{
		"name": "test-pod",
		"analysis": map[string]interface{}{
			"pod_phase":          "Running",
			"events_count":       float64(3),
			"log_lines_analyzed": float64(50),
			"ai_analysis":       `{"summary":"Pod is healthy.","severity":"LOW","errors_detected":[],"root_cause":"None","recommended_actions":["Continue monitoring"]}`,
		},
	}
	if err := PrintAnalysis(&buf, data, "test-ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"POD ANALYSIS", "test-pod", "test-ns", "AI ANALYSIS", "LOW", "Pod is healthy"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestPrintAnalysis_FallbackForNonJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]interface{}{
		"name": "test-pod",
		"analysis": map[string]interface{}{
			"pod_phase":    "CrashLoopBackOff",
			"ai_analysis":  "The pod is crashing because of an OOM error.",
			"events_count": float64(0),
		},
	}
	if err := PrintAnalysis(&buf, data, "ns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OOM error") {
		t.Error("expected raw analysis text in fallback output")
	}
}
