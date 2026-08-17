package domain

import (
	"fmt"
	"sort"
	"strings"
)

type FieldType string

const (
	TypeString    FieldType = "string"
	TypeInteger   FieldType = "integer"
	TypeDecimal   FieldType = "decimal"
	TypeBoolean   FieldType = "boolean"
	TypeTimestamp FieldType = "timestamp"
	TypeDate      FieldType = "date"
)

type ColumnSpec struct {
	Name     string    `json:"name"`
	Type     FieldType `json:"type"`
	Nullable bool      `json:"nullable"`
	Length   int       `json:"length"`
	Comment  string    `json:"comment"`
	Default  string    `json:"default"`
}

type IndexSpec struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Comment string   `json:"comment"`
}

type TableSpec struct {
	Name        string       `json:"name"`
	Comment     string       `json:"comment"`
	Columns     []ColumnSpec `json:"columns"`
	Indexes     []IndexSpec  `json:"indexes"`
	PrimaryKey  []string     `json:"primary_key"`
	SourceAlias string       `json:"source_alias"`
}

type SourceCatalog struct {
	Name      string      `json:"name"`
	Tables    []TableSpec `json:"tables"`
	Collected string      `json:"collected"`
}

func (c SourceCatalog) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("catalog name is required")
	}
	if len(c.Tables) == 0 {
		return fmt.Errorf("catalog must contain tables")
	}
	seen := make(map[string]bool)
	for _, table := range c.Tables {
		if err := table.Validate(); err != nil {
			return fmt.Errorf("table %s: %w", table.Name, err)
		}
		key := strings.ToLower(table.Name)
		if seen[key] {
			return fmt.Errorf("duplicate table %s", table.Name)
		}
		seen[key] = true
	}
	return nil
}

func (t TableSpec) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("table name is required")
	}
	if len(t.Columns) == 0 {
		return fmt.Errorf("table needs columns")
	}
	columns := map[string]bool{}
	for _, column := range t.Columns {
		if strings.TrimSpace(column.Name) == "" {
			return fmt.Errorf("column name is required")
		}
		if !validType(column.Type) {
			return fmt.Errorf("unsupported type %q", column.Type)
		}
		key := strings.ToLower(column.Name)
		if columns[key] {
			return fmt.Errorf("duplicate column %s", column.Name)
		}
		columns[key] = true
	}
	for _, key := range t.PrimaryKey {
		if !columns[strings.ToLower(key)] {
			return fmt.Errorf("primary key column %s missing", key)
		}
	}
	for _, index := range t.Indexes {
		if index.Name == "" || len(index.Columns) == 0 {
			return fmt.Errorf("index requires name and columns")
		}
		for _, column := range index.Columns {
			if !columns[strings.ToLower(column)] {
				return fmt.Errorf("index column %s missing", column)
			}
		}
	}
	return nil
}

func validType(t FieldType) bool {
	switch t {
	case TypeString, TypeInteger, TypeDecimal, TypeBoolean, TypeTimestamp, TypeDate:
		return true
	default:
		return false
	}
}

func (c SourceCatalog) Table(name string) (TableSpec, bool) {
	for _, table := range c.Tables {
		if strings.EqualFold(table.Name, name) {
			return table, true
		}
	}
	return TableSpec{}, false
}

func (c SourceCatalog) SortedTables() []TableSpec {
	result := append([]TableSpec(nil), c.Tables...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (t TableSpec) SortedColumns() []ColumnSpec {
	result := append([]ColumnSpec(nil), t.Columns...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
