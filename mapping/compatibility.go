package mapping

import (
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"
)

type CompatibilityLevel string

const (
	CompatibilityExact  CompatibilityLevel = "exact"
	CompatibilitySafe   CompatibilityLevel = "safe"
	CompatibilityLossy  CompatibilityLevel = "lossy"
	CompatibilityManual CompatibilityLevel = "manual"
)

type ColumnMapping struct {
	Table       string             `json:"table"`
	Column      string             `json:"column"`
	SourceType  domain.FieldType   `json:"source_type"`
	TargetType  string             `json:"target_type"`
	Level       CompatibilityLevel `json:"level"`
	Reason      string             `json:"reason"`
	Declaration string             `json:"declaration"`
}

type IndexMapping struct {
	Table       string   `json:"table"`
	SourceName  string   `json:"source_name"`
	TargetName  string   `json:"target_name"`
	Columns     []string `json:"columns"`
	Unique      bool     `json:"unique"`
	Supported   bool     `json:"supported"`
	Explanation string   `json:"explanation"`
}

type CompatibilityReport struct {
	Dialect TargetDialect   `json:"dialect"`
	Columns []ColumnMapping `json:"columns"`
	Indexes []IndexMapping  `json:"indexes"`
	Exact   int             `json:"exact"`
	Safe    int             `json:"safe"`
	Lossy   int             `json:"lossy"`
	Manual  int             `json:"manual"`
	Ready   bool            `json:"ready"`
}

func AssessCompatibility(catalog domain.SourceCatalog, dialect TargetDialect) (CompatibilityReport, error) {
	if err := catalog.Validate(); err != nil {
		return CompatibilityReport{}, err
	}
	report := CompatibilityReport{Dialect: dialect, Ready: true}
	rules := RulesFor(dialect)
	for _, table := range catalog.SortedTables() {
		for _, column := range table.SortedColumns() {
			rule, ok := rules[column.Type]
			if !ok {
				mapping := ColumnMapping{Table: table.Name, Column: column.Name, SourceType: column.Type, Level: CompatibilityManual, Reason: "no type rule"}
				report.Columns = append(report.Columns, mapping)
				report.Manual++
				report.Ready = false
				continue
			}
			level, reason := classifyColumn(column, dialect, rule)
			mapping := ColumnMapping{Table: table.Name, Column: column.Name, SourceType: column.Type, TargetType: rule.Target, Level: level, Reason: reason, Declaration: rule.Declaration}
			report.Columns = append(report.Columns, mapping)
			switch level {
			case CompatibilityExact:
				report.Exact++
			case CompatibilitySafe:
				report.Safe++
			case CompatibilityLossy:
				report.Lossy++
			case CompatibilityManual:
				report.Manual++
				report.Ready = false
			}
		}
		for _, index := range table.Indexes {
			report.Indexes = append(report.Indexes, assessIndex(table, index, dialect))
		}
	}
	sort.SliceStable(report.Columns, func(i, j int) bool {
		if report.Columns[i].Table != report.Columns[j].Table {
			return report.Columns[i].Table < report.Columns[j].Table
		}
		return report.Columns[i].Column < report.Columns[j].Column
	})
	return report, nil
}

func classifyColumn(column domain.ColumnSpec, dialect TargetDialect, rule TypeRule) (CompatibilityLevel, string) {
	if rule.Warning != "" {
		return CompatibilityLossy, rule.Warning
	}
	if column.Type == domain.TypeString && column.Length > 0 {
		if dialect == DialectPostgres || dialect == DialectMySQL {
			return CompatibilityExact, "bounded string length preserved"
		}
		return CompatibilitySafe, "target accepts the source string length"
	}
	if column.Type == domain.TypeDecimal && dialect == DialectSQLite {
		return CompatibilitySafe, "numeric affinity preserves decimal values"
	}
	if column.Default != "" {
		upper := strings.ToUpper(column.Default)
		if strings.Contains(upper, "CURRENT_TIMESTAMP") && dialect == DialectSQLite {
			return CompatibilitySafe, "current timestamp syntax supported"
		}
		if strings.Contains(upper, "NEXTVAL") && dialect != DialectPostgres {
			return CompatibilityManual, "sequence default requires manual mapping"
		}
	}
	return CompatibilityExact, "direct type mapping"
}

func assessIndex(table domain.TableSpec, index domain.IndexSpec, dialect TargetDialect) IndexMapping {
	result := IndexMapping{Table: table.Name, SourceName: index.Name, TargetName: NormalizeName(index.Name), Columns: append([]string(nil), index.Columns...), Unique: index.Unique, Supported: true, Explanation: "index can be recreated"}
	if len(index.Columns) > 16 && dialect == DialectMySQL {
		result.Supported = false
		result.Explanation = "target index column limit exceeded"
	}
	if strings.TrimSpace(index.Name) == "" {
		result.Supported = false
		result.Explanation = "source index name is empty"
	}
	return result
}

func (r CompatibilityReport) BlockingMappings() []ColumnMapping {
	result := []ColumnMapping{}
	for _, mapping := range r.Columns {
		if mapping.Level == CompatibilityManual {
			result = append(result, mapping)
		}
	}
	return result
}

func (r CompatibilityReport) WarningMappings() []ColumnMapping {
	result := []ColumnMapping{}
	for _, mapping := range r.Columns {
		if mapping.Level == CompatibilityLossy || mapping.Level == CompatibilityManual {
			result = append(result, mapping)
		}
	}
	return result
}

func (r CompatibilityReport) SummaryText() string {
	state := "ready"
	if !r.Ready {
		state = "requires review"
	}
	return fmt.Sprintf("%s: %d exact, %d safe, %d lossy, %d manual", state, r.Exact, r.Safe, r.Lossy, r.Manual)
}

func (r CompatibilityReport) TableLevels() map[string]CompatibilityLevel {
	levels := map[string]CompatibilityLevel{}
	priority := map[CompatibilityLevel]int{CompatibilityExact: 0, CompatibilitySafe: 1, CompatibilityLossy: 2, CompatibilityManual: 3}
	for _, column := range r.Columns {
		current, ok := levels[column.Table]
		if !ok || priority[column.Level] > priority[current] {
			levels[column.Table] = column.Level
		}
	}
	return levels
}
