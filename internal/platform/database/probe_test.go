package database

import (
	"context"
	"strings"
	"testing"
)

func TestDatabaseConfigurationFailsClosed(t *testing.T) {
	valid := Config{Host: "postgres", Port: 5432, Database: "atlas_local", User: "atlas_api", Password: strings.Repeat("a", 32)}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	tests := []Config{
		{},
		{Host: "postgres", Port: 5432, Database: "atlas-local", User: "atlas_api", Password: strings.Repeat("a", 32)},
		{Host: "postgres@real.example", Port: 5432, Database: "atlas_local", User: "atlas_api", Password: strings.Repeat("a", 32)},
		{Host: "postgres", Port: 5432, Database: "atlas_local", User: "atlas_api", Password: "short"},
	}
	for _, config := range tests {
		if err := config.Validate(); err == nil {
			t.Fatal("unsafe database readiness configuration was accepted")
		}
	}
}

func TestSchemaProbeFailsClosedWithoutDatabase(t *testing.T) {
	probe, err := NewSchemaProbe(context.Background(), Config{
		Host: "127.0.0.1", Port: 1, Database: "atlas_local", User: "atlas_api", Password: strings.Repeat("a", 32),
	}, ProbeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	if probe.Ready(context.Background()) {
		t.Fatal("unreachable schema was reported current")
	}
	if (*SchemaProbe)(nil).Ready(context.Background()) {
		t.Fatal("nil schema probe was reported current")
	}
}
