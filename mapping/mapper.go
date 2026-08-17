package mapping

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"
)

type Mapper struct {
	Dialect TargetDialect
	Policy  CommentPolicy
}

func NewMapper(dialect TargetDialect) Mapper {
	return Mapper{Dialect: dialect, Policy: DefaultCommentPolicy()}
}

func (m Mapper) Preview(catalog domain.SourceCatalog) (domain.Preview, error) {
	if err := catalog.Validate(); err != nil {
		return domain.Preview{}, err
	}
	result := domain.Preview{Summary: domain.MappingSummary{Tables: len(catalog.Tables)}}
	rules := RulesFor(m.Dialect)
	for _, table := range catalog.SortedTables() {
		statement, warnings, err := m.mapTable(table, rules)
		if err != nil {
			return domain.Preview{}, err
		}
		result.Statements = append(result.Statements, statement)
		result.Warnings = append(result.Warnings, warnings...)
		result.Summary.Columns += len(table.Columns)
		result.Summary.Indexes += len(table.Indexes)
	}
	result.Summary.Warnings = len(result.Warnings)
	return result, nil
}

func (m Mapper) mapTable(table domain.TableSpec, rules map[domain.FieldType]TypeRule) (domain.DDLStatement, []string, error) {
	columns := table.SortedColumns()
	parts := make([]string, 0, len(columns)+2)
	warnings := []string{}
	for _, column := range columns {
		rule, ok := rules[column.Type]
		if !ok {
			return domain.DDLStatement{}, nil, fmt.Errorf("no rule for %s", column.Type)
		}
		typeName := rule.Declaration
		if column.Length > 0 && column.Type == domain.TypeString && m.Dialect == DialectPostgres {
			typeName = fmt.Sprintf("VARCHAR(%d)", column.Length)
		}
		nullability := " NULL"
		if !column.Nullable {
			nullability = " NOT NULL"
		}
		defaultSQL := ""
		if column.Default != "" {
			defaultSQL = " DEFAULT " + column.Default
		}
		parts = append(parts, QuoteIdentifier(NormalizeName(column.Name), m.Dialect)+" "+typeName+nullability+defaultSQL+RenderComment(ColumnComment(column, m.Policy), string(m.Dialect)))
		if rule.Warning != "" {
			warnings = append(warnings, table.Name+"."+column.Name+": "+rule.Warning)
		}
	}
	for _, key := range table.PrimaryKey {
		parts = append(parts, "PRIMARY KEY ("+m.columnsSQL(key)+")")
	}
	for _, index := range table.Indexes {
		if index.Unique {
			warnings = append(warnings, table.Name+"."+index.Name+": unique index preserved")
		}
	}
	name := NormalizeName(table.Name)
	sql := "CREATE TABLE " + QuoteIdentifier(name, m.Dialect) + " (" + strings.Join(parts, ", ") + ");"
	return domain.DDLStatement{Kind: "table", Table: name, SQL: sql, Warnings: append([]string(nil), warnings...), SourceHash: hashTable(table)}, warnings, nil
}

func (m Mapper) columnsSQL(name string) string {
	return QuoteIdentifier(NormalizeName(name), m.Dialect)
}

func hashTable(table domain.TableSpec) string {
	value := table.Name
	for _, column := range table.SortedColumns() {
		value += "|" + column.Name + ":" + string(column.Type)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func (m Mapper) IndexStatements(catalog domain.SourceCatalog) []domain.DDLStatement {
	statements := []domain.DDLStatement{}
	for _, table := range catalog.SortedTables() {
		indexes := append([]domain.IndexSpec(nil), table.Indexes...)
		sort.Slice(indexes, func(i, j int) bool { return indexes[i].Name < indexes[j].Name })
		for _, index := range indexes {
			kind := "CREATE INDEX"
			if index.Unique {
				kind = "CREATE UNIQUE INDEX"
			}
			columns := make([]string, 0, len(index.Columns))
			for _, column := range index.Columns {
				columns = append(columns, m.columnsSQL(column))
			}
			sql := fmt.Sprintf("%s %s ON %s (%s);", kind, QuoteIdentifier(NormalizeName(index.Name), m.Dialect), QuoteIdentifier(NormalizeName(table.Name), m.Dialect), strings.Join(columns, ", "))
			statements = append(statements, domain.DDLStatement{Kind: "index", Table: table.Name, SQL: sql, Warnings: []string{IndexComment(index, m.Policy)}})
		}
	}
	return statements
}
