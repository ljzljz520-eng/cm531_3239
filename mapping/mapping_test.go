package mapping

import (
	"strings"
	"testing"

	"example.com/mestransform/domain"
)

func TestMapperProducesDDLAndIndexes(t *testing.T) {
	catalog := domain.SourceCatalog{
		Name: "mes",
		Tables: []domain.TableSpec{{
			Name:       "Work Orders",
			Columns:    []domain.ColumnSpec{{Name: "id", Type: domain.TypeString}, {Name: "quantity", Type: domain.TypeInteger}},
			PrimaryKey: []string{"id"},
			Indexes:    []domain.IndexSpec{{Name: "idx_quantity", Columns: []string{"quantity"}}},
		}},
	}
	mapper := NewMapper(DialectPostgres)
	preview, err := mapper.Preview(catalog)
	if err != nil {
		t.Fatal(err)
	}
	indexes := mapper.IndexStatements(catalog)
	if len(preview.Statements) != 1 || len(indexes) != 1 {
		t.Fatalf("unexpected statement count %d %d", len(preview.Statements), len(indexes))
	}
	if !strings.Contains(preview.Statements[0].SQL, `"work_orders"`) {
		t.Fatal(preview.Statements[0].SQL)
	}
}

func TestDialectRulesExposeWarnings(t *testing.T) {
	rules := RulesFor(DialectSQLite)
	if rules[domain.TypeTimestamp].Warning == "" {
		t.Fatal("timestamp warning missing")
	}
	if _, err := DialectFor("oracle"); err == nil {
		t.Fatal("unsupported dialect accepted")
	}
}
