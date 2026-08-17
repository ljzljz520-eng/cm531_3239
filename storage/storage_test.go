package storage

import (
	"path/filepath"
	"testing"

	"example.com/mestransform/domain"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mes.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(store)
	record := domain.ConversionRecord{ID: "conversion-1", IdempotencyKey: "persist-key", Status: "completed", Sequence: 1}
	if err := repo.SaveRecord(record); err != nil {
		t.Fatal(err)
	}
	request := domain.ConversionRequest{WorkOrders: []domain.WorkOrder{{ID: "w1"}}, Equipment: []domain.Equipment{{ID: "e1"}}, Inspections: []domain.QualityInspection{{ID: "q1"}}, Materials: []domain.Material{{ID: "m1"}}}
	if err := repo.SaveEntities(request); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	repo = NewRepository(reopened)
	got, found, err := repo.RecordByKey("persist-key")
	if err != nil || !found || got.ID != record.ID {
		t.Fatalf("record not recovered %#v %v %v", got, found, err)
	}
	counts, err := repo.EntityCounts()
	if err != nil {
		t.Fatal(err)
	}
	for name, count := range counts {
		if count != 1 {
			t.Fatalf("entity %s count %d", name, count)
		}
	}
}

func TestSequenceContinues(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "seq.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repo := NewRepository(store)
	first, _ := repo.NextSequence()
	second, _ := repo.NextSequence()
	if first != 1 || second != 2 {
		t.Fatalf("sequences %d %d", first, second)
	}
}
