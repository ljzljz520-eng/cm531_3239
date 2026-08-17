package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
	"example.com/mestransform/service"
	"example.com/mestransform/storage"
)

func main() {
	dbPath := flag.String("db", "/tmp/mestransform-cli.db", "embedded bbolt database path")
	dialectName := flag.String("dialect", "sqlite", "target dialect")
	flag.Parse()
	dialect, err := mapping.DialectFor(*dialectName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	store, err := storage.Open(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer store.Close()
	repo := storage.NewRepository(store)
	conversion := service.NewConversionService(repo, dialect)
	request := sampleRequest()
	preview, err := conversion.Preview(context.Background(), request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, statement := range preview.Statements {
		fmt.Println(statement.SQL)
	}
}

func sampleRequest() domain.ConversionRequest {
	return domain.ConversionRequest{
		IdempotencyKey: "cli-preview", SourceName: "mes-source", TargetName: "sqlite", RequestedBy: "cli",
		Catalog: domain.SourceCatalog{Name: "sample", Tables: []domain.TableSpec{{Name: "work_orders", Comment: "工单", Columns: []domain.ColumnSpec{{Name: "id", Type: domain.TypeString, Nullable: false, Comment: "主键"}, {Name: "quantity", Type: domain.TypeInteger, Nullable: false}}, PrimaryKey: []string{"id"}}}}, DryRun: true,
	}
}
