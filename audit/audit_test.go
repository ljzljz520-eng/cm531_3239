package audit

import (
	"path/filepath"
	"testing"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

func TestAuditQueryAndTimeline(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repo := storage.NewRepository(store)
	record := domain.ConversionRecord{ID: "conversion-1", RequestedBy: "planner", TargetName: "sqlite", Status: "completed", Sequence: 1, Summary: domain.MappingSummary{Tables: 4}}
	if err := repo.SaveRecord(record); err != nil {
		t.Fatal(err)
	}
	logger := NewLogger(repo)
	if err := logger.RecordConversion(record); err != nil {
		t.Fatal(err)
	}
	timeline, err := logger.Timeline(record.ID)
	if err != nil || len(timeline) != 1 {
		t.Fatalf("timeline %#v %v", timeline, err)
	}
	query := NewQuery(repo)
	records, err := query.RecordsByRequester("PLANNER")
	if err != nil || len(records) != 1 {
		t.Fatalf("records %#v %v", records, err)
	}
}
