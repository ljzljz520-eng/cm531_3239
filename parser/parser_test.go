package parser

import (
	"strings"
	"testing"
)

func TestParseDDL(t *testing.T) {
	input := "CREATE TABLE work_orders (\nid TEXT NOT NULL,\nquantity INTEGER,\nPRIMARY KEY (id)\n)"
	catalog, err := New(true).Parse(strings.NewReader(input), FormatDDL)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tables) != 1 || len(catalog.Tables[0].Columns) != 2 {
		t.Fatalf("unexpected catalog %#v", catalog)
	}
}

func TestParseMaterialRows(t *testing.T) {
	input := "id,sku,name,unit,on_hand,reorder_at\nm1,STEEL,Steel,kg,90,20\n"
	rows, err := ParseRows(strings.NewReader(input), "material")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows.Materials) != 1 || rows.Materials[0].OnHand != 90 {
		t.Fatalf("unexpected rows %#v", rows)
	}
}
