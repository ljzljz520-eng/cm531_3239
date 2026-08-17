package domain

import "testing"

func TestCatalogValidation(t *testing.T) {
	catalog := SourceCatalog{Name: "mes", Tables: []TableSpec{{Name: "orders", Columns: []ColumnSpec{{Name: "id", Type: TypeString}}, PrimaryKey: []string{"id"}}}}
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, ok := catalog.Table("ORDERS"); !ok {
		t.Fatal("table lookup failed")
	}
}

func TestRequestValidationRejectsInvalidEntities(t *testing.T) {
	request := ConversionRequest{IdempotencyKey: "k", SourceName: "source", TargetName: "target", Catalog: SourceCatalog{Name: "mes", Tables: []TableSpec{{Name: "orders", Columns: []ColumnSpec{{Name: "id", Type: TypeString}}}}}, WorkOrders: []WorkOrder{{ID: "o", OrderNo: "WO", Quantity: 0}}}
	if err := request.Validate(); err == nil {
		t.Fatal("invalid request accepted")
	}
}
