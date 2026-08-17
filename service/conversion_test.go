package service

import (
	"context"
	"path/filepath"
	"testing"

	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
	"example.com/mestransform/storage"
)

func testService(t *testing.T) (*ConversionService, *storage.Repository) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	repo := storage.NewRepository(store)
	return NewConversionService(repo, mapping.DialectSQLite), repo
}

func testRequest(key string) domain.ConversionRequest {
	return domain.ConversionRequest{
		IdempotencyKey: key, SourceName: "legacy-mes", TargetName: "sqlite", RequestedBy: "planner",
		Catalog: domain.SourceCatalog{Name: "factory", Tables: []domain.TableSpec{
			{Name: "work_orders", Comment: "work orders", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "quantity", Type: domain.TypeInteger}}, PrimaryKey: []string{"id"}, Indexes: []domain.IndexSpec{{Name: "idx_order_qty", Columns: []string{"quantity"}}}},
			{Name: "equipment", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "capacity", Type: domain.TypeInteger}}, PrimaryKey: []string{"id"}},
		}},
		WorkOrders:  []domain.WorkOrder{{ID: "w1", OrderNo: "WO-1", Product: "panel", Quantity: 8}},
		Equipment:   []domain.Equipment{{ID: "e1", Code: "CUT-1", Name: "cutter", Capacity: 10}},
		Inspections: []domain.QualityInspection{{ID: "q1", OrderID: "w1", Metric: "width", Expected: "10", Actual: "10", Passed: true}},
		Materials:   []domain.Material{{ID: "m1", SKU: "STEEL", Name: "steel", Unit: "kg", OnHand: 100, ReorderAt: 20}},
	}
}

func TestPreviewAndConvert(t *testing.T) {
	svc, repo := testService(t)
	preview, err := svc.Preview(context.Background(), testRequest("preview-key"))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Summary.Tables != 2 || len(preview.Statements) != 3 {
		t.Fatalf("preview %#v", preview)
	}
	record, err := svc.Convert(context.Background(), testRequest("execute-key"))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repo.RecordByID(record.ID)
	if err != nil || stored.Status != "completed" {
		t.Fatalf("stored %#v %v", stored, err)
	}
}

func TestRepeatedRequestReturnsStoredResult(t *testing.T) {
	svc, repo := testService(t)
	first, err := svc.Convert(context.Background(), testRequest("same-key"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Convert(context.Background(), testRequest("same-key"))
	if err != nil {
		t.Fatal(err)
	}
	records, err := repo.ListRecords()
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || len(records) != 1 {
		t.Fatalf("first=%s second=%s records=%d", first.ID, second.ID, len(records))
	}
}
