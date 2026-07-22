package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type metricCatalog struct {
	Version           int `json:"version"`
	CardinalityBudget int `json:"cardinality_budget_per_metric"`
	Metrics           []struct {
		Name   string              `json:"name"`
		Kind   string              `json:"kind"`
		Status string              `json:"status"`
		Owner  string              `json:"owner"`
		Labels map[string][]string `json:"labels"`
	} `json:"metrics"`
	Alerts []struct {
		ID, Metric, Severity, Owner, Condition, Rationale, Runbook, Test string
	} `json:"alerts"`
	Dashboards []struct {
		Name, Owner string
		Panels      []string `json:"panels"`
	} `json:"dashboards"`
}

func loadCatalog(t *testing.T) (metricCatalog, string) {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	contents, err := os.ReadFile(filepath.Join(root, "deploy", "observability", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog metricCatalog
	if err := json.Unmarshal(contents, &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog, root
}

func TestMetricCatalogEnforcesCardinalityAndRuntimeCoverage(t *testing.T) {
	catalog, root := loadCatalog(t)
	if catalog.Version != 1 || catalog.CardinalityBudget < 1 || catalog.CardinalityBudget > 128 {
		t.Fatal("metric catalog policy is invalid")
	}
	required := map[string]string{
		"http.server.request.count": "emitted", "http.server.request.duration": "emitted",
		"atlas.database.readiness.count": "emitted", "atlas.database.readiness.duration": "emitted",
		"atlas.database.pool.connections": "emitted", "atlas.build.info": "emitted",
		"atlas.queue.lag": "definition-only", "atlas.worker.retry.count": "definition-only",
	}
	seen := make(map[string]struct{}, len(catalog.Metrics))
	for _, metric := range catalog.Metrics {
		if _, duplicate := seen[metric.Name]; duplicate || metric.Owner == "" || metric.Kind == "" || required[metric.Name] != metric.Status {
			t.Fatalf("invalid metric catalog entry %q", metric.Name)
		}
		seen[metric.Name] = struct{}{}
		cardinality := 1
		for label, values := range metric.Labels {
			lower := strings.NewReplacer(".", "_", "-", "_").Replace(strings.ToLower(label))
			for _, forbidden := range []string{"request_id", "correlation_id", "trace_id", "tenant", "actor", "user", "email", "account"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("high-cardinality identity label %q is forbidden", label)
				}
			}
			if len(values) == 0 || len(values) > 16 {
				t.Fatalf("label %q has an invalid allowlist", label)
			}
			cardinality *= len(values)
		}
		if cardinality > catalog.CardinalityBudget {
			t.Fatalf("metric %s cardinality %d exceeds budget", metric.Name, cardinality)
		}
		delete(required, metric.Name)
	}
	if len(required) != 0 {
		t.Fatalf("required metrics missing: %v", required)
	}

	serverSource, err := os.ReadFile(filepath.Join(root, "cmd", "api", "internal", "server", "server.go"))
	if err != nil {
		t.Fatal(err)
	}
	databaseSource, err := os.ReadFile(filepath.Join(root, "internal", "platform", "database", "probe.go"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := os.ReadFile(filepath.Join(root, "internal", "platform", "telemetry", "runtime.go"))
	if err != nil {
		t.Fatal(err)
	}
	sources := string(serverSource) + string(databaseSource) + string(runtimeSource)
	for name := range seen {
		status := ""
		for _, metric := range catalog.Metrics {
			if metric.Name == name {
				status = metric.Status
			}
		}
		contains := strings.Contains(sources, `"`+name+`"`)
		if status == "emitted" && !contains {
			t.Fatalf("emitted metric %s is not implemented", name)
		}
		if status == "definition-only" && contains {
			t.Fatalf("future metric %s was emitted before its subsystem exists", name)
		}
	}
}

func TestAlertAndDashboardCatalogIsOwnedAndRunnable(t *testing.T) {
	catalog, root := loadCatalog(t)
	metrics := make(map[string]struct{}, len(catalog.Metrics))
	for _, metric := range catalog.Metrics {
		metrics[metric.Name] = struct{}{}
	}
	alerts := make(map[string]struct{}, len(catalog.Alerts))
	for _, alert := range catalog.Alerts {
		if _, duplicate := alerts[alert.ID]; duplicate || alert.ID == "" || alert.Owner == "" || alert.Condition == "" || alert.Rationale == "" || alert.Test == "" {
			t.Fatalf("alert metadata is incomplete for %s", alert.ID)
		}
		if alert.Severity != "page" && alert.Severity != "ticket" {
			t.Fatalf("alert %s has an invalid severity", alert.ID)
		}
		if _, found := metrics[alert.Metric]; !found {
			t.Fatalf("alert %s references unknown metric", alert.ID)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(alert.Runbook))); err != nil {
			t.Fatalf("alert %s runbook: %v", alert.ID, err)
		}
		alerts[alert.ID] = struct{}{}
	}
	for _, dashboard := range catalog.Dashboards {
		if dashboard.Name == "" || dashboard.Owner == "" || len(dashboard.Panels) == 0 {
			t.Fatal("dashboard metadata is incomplete")
		}
		for _, panel := range dashboard.Panels {
			if _, found := metrics[panel]; !found {
				t.Fatalf("dashboard references unknown metric %s", panel)
			}
		}
	}
}
