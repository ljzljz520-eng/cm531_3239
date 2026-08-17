package mestransform_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"example.com/mestransform/audit"
	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
	"example.com/mestransform/service"
	"example.com/mestransform/storage"
)

func integrationRequest(key string) domain.ConversionRequest {
	tables := []domain.TableSpec{
		{Name: "work_orders", Comment: "manufacturing orders", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "order_no", Type: domain.TypeString}, {Name: "quantity", Type: domain.TypeInteger}}, PrimaryKey: []string{"id"}, Indexes: []domain.IndexSpec{{Name: "idx_order_no", Columns: []string{"order_no"}, Unique: true}}},
		{Name: "equipment", Comment: "factory equipment", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "code", Type: domain.TypeString}, {Name: "capacity", Type: domain.TypeInteger}}, PrimaryKey: []string{"id"}},
		{Name: "quality_inspections", Comment: "inspection results", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "passed", Type: domain.TypeBoolean}, {Name: "actual", Type: domain.TypeDecimal}}, PrimaryKey: []string{"id"}},
		{Name: "materials", Comment: "production material", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false}, {Name: "sku", Type: domain.TypeString}, {Name: "on_hand", Type: domain.TypeInteger}}, PrimaryKey: []string{"id"}},
	}
	return domain.ConversionRequest{
		IdempotencyKey: key, SourceName: "legacy-mes", TargetName: "sqlite", RequestedBy: "operator", Catalog: domain.SourceCatalog{Name: "plant-one", Tables: tables},
		WorkOrders:  []domain.WorkOrder{{ID: "wo-1", OrderNo: "WO-2026-1", Product: "panel", Quantity: 20, Status: "released"}},
		Equipment:   []domain.Equipment{{ID: "eq-1", Code: "PRESS-1", Name: "press", Line: "A", State: "ready", Capacity: 50}},
		Inspections: []domain.QualityInspection{{ID: "qi-1", OrderID: "wo-1", Metric: "width", Expected: "100", Actual: "100", Passed: true}},
		Materials:   []domain.Material{{ID: "ma-1", SKU: "AL-01", Name: "aluminium", Unit: "kg", OnHand: 400, ReorderAt: 100}},
	}
}

func openSystem(t *testing.T, path string) (*storage.Store, *storage.Repository, *service.ConversionService) {
	t.Helper()
	store, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := storage.NewRepository(store)
	return store, repo, service.NewConversionService(repo, mapping.DialectSQLite)
}

func TestWorkflowPreviewExecuteAudit(t *testing.T) {
	store, repo, conversion := openSystem(t, filepath.Join(t.TempDir(), "workflow.db"))
	defer store.Close()
	request := integrationRequest("workflow-preview")
	previewService := service.NewPreviewService(conversion)
	ddl, err := previewService.DDL(context.Background(), request)
	if err != nil || !strings.Contains(ddl, "CREATE TABLE") {
		t.Fatalf("ddl=%q err=%v", ddl, err)
	}
	record, err := previewService.Confirm(context.Background(), request, true)
	if err != nil {
		t.Fatal(err)
	}
	logger := audit.NewLogger(repo)
	if err := logger.RecordConversion(record); err != nil {
		t.Fatal(err)
	}
	timeline, err := logger.Timeline(record.ID)
	if err != nil || len(timeline) != 2 {
		t.Fatalf("timeline=%#v err=%v", timeline, err)
	}
}

func TestWorkflowBatchPersistQuery(t *testing.T) {
	store, repo, conversion := openSystem(t, filepath.Join(t.TempDir(), "batch.db"))
	defer store.Close()
	requests := []domain.ConversionRequest{integrationRequest("workflow-batch-1"), integrationRequest("workflow-batch-2")}
	requests[1].WorkOrders[0].ID = "wo-2"
	requests[1].WorkOrders[0].OrderNo = "WO-2026-2"
	requests[1].Equipment[0].ID = "eq-2"
	requests[1].Inspections[0].ID = "qi-2"
	requests[1].Inspections[0].OrderID = "wo-2"
	requests[1].Materials[0].ID = "ma-2"
	outcome, err := conversion.RunBatch(context.Background(), "daily-migration", requests, 2)
	if err != nil || outcome.Job.Processed != 2 {
		t.Fatalf("outcome=%#v err=%v", outcome, err)
	}
	job, err := repo.BatchByID(outcome.Job.ID)
	if err != nil || job.Status != "completed" {
		t.Fatalf("job=%#v err=%v", job, err)
	}
	query := audit.NewQuery(repo)
	records, err := query.RecordsByStatus("completed")
	if err != nil || len(records) != 2 {
		t.Fatalf("records=%#v err=%v", records, err)
	}
}

func TestWorkflowExecuteDownloadReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	store, repo, conversion := openSystem(t, path)
	record, err := conversion.Convert(context.Background(), integrationRequest("workflow-download"))
	if err != nil {
		t.Fatal(err)
	}
	history := service.NewHistoryService(repo)
	download, err := history.Download(record.ID, "text")
	if err != nil || download.Payload == "" {
		t.Fatalf("download=%#v err=%v", download, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, reopenedRepo, _ := openSystem(t, path)
	defer reopened.Close()
	stored, err := reopenedRepo.DownloadByID(download.ID)
	if err != nil || stored.RecordID != record.ID {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
	counts, err := audit.NewQuery(reopenedRepo).EntitySummary()
	if err != nil || len(counts) != 4 {
		t.Fatalf("counts=%#v err=%v", counts, err)
	}
}
