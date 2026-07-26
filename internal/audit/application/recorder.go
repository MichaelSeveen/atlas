// Package application exposes Audit-owned application composition boundaries.
package application

import (
	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/audit/persistence"
)

// NewRecorder returns the append-only PostgreSQL adapter behind the Audit boundary.
func NewRecorder() audit.Recorder {
	return persistence.Recorder{}
}
