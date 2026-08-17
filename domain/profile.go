package domain

import (
	"fmt"
	"sort"
	"strings"
)

type SourceVendor string

const (
	VendorPostgres  SourceVendor = "postgres"
	VendorMySQL     SourceVendor = "mysql"
	VendorSQLite    SourceVendor = "sqlite"
	VendorSQLServer SourceVendor = "sqlserver"
)

type ConnectionProfile struct {
	Name       string            `json:"name"`
	Vendor     SourceVendor      `json:"vendor"`
	Database   string            `json:"database"`
	Schema     string            `json:"schema"`
	Properties map[string]string `json:"properties"`
	ReadOnly   bool              `json:"read_only"`
}

type CatalogStatistics struct {
	TableCount       int               `json:"table_count"`
	ColumnCount      int               `json:"column_count"`
	IndexCount       int               `json:"index_count"`
	RequiredColumns  int               `json:"required_columns"`
	NullableColumns  int               `json:"nullable_columns"`
	TypeDistribution map[FieldType]int `json:"type_distribution"`
	LargestTable     string            `json:"largest_table"`
	LargestWidth     int               `json:"largest_width"`
}

type CatalogIssue struct {
	Level   string `json:"level"`
	Table   string `json:"table"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

func (p ConnectionProfile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("connection profile name is required")
	}
	if strings.TrimSpace(p.Database) == "" {
		return fmt.Errorf("connection database is required")
	}
	switch p.Vendor {
	case VendorPostgres, VendorMySQL, VendorSQLite, VendorSQLServer:
	default:
		return fmt.Errorf("unsupported source vendor %q", p.Vendor)
	}
	if p.Vendor != VendorSQLite && strings.TrimSpace(p.Schema) == "" {
		return fmt.Errorf("schema is required for %s", p.Vendor)
	}
	for key := range p.Properties {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("connection property key cannot be empty")
		}
	}
	return nil
}

func (p ConnectionProfile) SafeProperties() map[string]string {
	result := make(map[string]string, len(p.Properties))
	for key, value := range p.Properties {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") {
			result[key] = "***"
		} else {
			result[key] = value
		}
	}
	return result
}

func (c SourceCatalog) Statistics() CatalogStatistics {
	stats := CatalogStatistics{TableCount: len(c.Tables), TypeDistribution: map[FieldType]int{}}
	for _, table := range c.Tables {
		stats.ColumnCount += len(table.Columns)
		stats.IndexCount += len(table.Indexes)
		if len(table.Columns) > stats.LargestWidth {
			stats.LargestWidth = len(table.Columns)
			stats.LargestTable = table.Name
		}
		for _, column := range table.Columns {
			stats.TypeDistribution[column.Type]++
			if column.Nullable {
				stats.NullableColumns++
			} else {
				stats.RequiredColumns++
			}
		}
	}
	return stats
}

func (c SourceCatalog) Inspect() []CatalogIssue {
	issues := []CatalogIssue{}
	for _, table := range c.Tables {
		if strings.TrimSpace(table.Comment) == "" {
			issues = append(issues, CatalogIssue{Level: "info", Table: table.Name, Message: "table comment is missing"})
		}
		if len(table.PrimaryKey) == 0 {
			issues = append(issues, CatalogIssue{Level: "warning", Table: table.Name, Message: "primary key is missing"})
		}
		indexed := map[string]bool{}
		for _, index := range table.Indexes {
			for _, column := range index.Columns {
				indexed[strings.ToLower(column)] = true
			}
		}
		for _, column := range table.Columns {
			if strings.TrimSpace(column.Comment) == "" {
				issues = append(issues, CatalogIssue{Level: "info", Table: table.Name, Column: column.Name, Message: "column comment is missing"})
			}
			if column.Type == TypeString && column.Length == 0 {
				issues = append(issues, CatalogIssue{Level: "info", Table: table.Name, Column: column.Name, Message: "unbounded string column"})
			}
			if strings.HasSuffix(strings.ToLower(column.Name), "_id") && !indexed[strings.ToLower(column.Name)] {
				issues = append(issues, CatalogIssue{Level: "warning", Table: table.Name, Column: column.Name, Message: "identifier column is not indexed"})
			}
		}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Table != issues[j].Table {
			return issues[i].Table < issues[j].Table
		}
		if issues[i].Column != issues[j].Column {
			return issues[i].Column < issues[j].Column
		}
		return issues[i].Message < issues[j].Message
	})
	return issues
}

func (c SourceCatalog) SelectTables(names []string) SourceCatalog {
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[strings.ToLower(strings.TrimSpace(name))] = true
	}
	selected := SourceCatalog{Name: c.Name, Collected: c.Collected}
	for _, table := range c.Tables {
		if wanted[strings.ToLower(table.Name)] {
			selected.Tables = append(selected.Tables, table)
		}
	}
	return selected
}

func (c SourceCatalog) RenameTables(renames map[string]string) SourceCatalog {
	result := SourceCatalog{Name: c.Name, Collected: c.Collected, Tables: append([]TableSpec(nil), c.Tables...)}
	for index := range result.Tables {
		for source, target := range renames {
			if strings.EqualFold(result.Tables[index].Name, source) {
				result.Tables[index].SourceAlias = result.Tables[index].Name
				result.Tables[index].Name = target
				break
			}
		}
	}
	return result
}

func (c SourceCatalog) Merge(other SourceCatalog) (SourceCatalog, error) {
	result := SourceCatalog{Name: c.Name, Collected: c.Collected, Tables: append([]TableSpec(nil), c.Tables...)}
	seen := map[string]bool{}
	for _, table := range result.Tables {
		seen[strings.ToLower(table.Name)] = true
	}
	for _, table := range other.Tables {
		key := strings.ToLower(table.Name)
		if seen[key] {
			return SourceCatalog{}, fmt.Errorf("cannot merge duplicate table %s", table.Name)
		}
		seen[key] = true
		result.Tables = append(result.Tables, table)
	}
	if err := result.Validate(); err != nil {
		return SourceCatalog{}, err
	}
	return result, nil
}
