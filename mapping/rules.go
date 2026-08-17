package mapping

import (
	"fmt"
	"strings"

	"example.com/mestransform/domain"
)

type TargetDialect string

const (
	DialectPostgres TargetDialect = "postgres"
	DialectMySQL    TargetDialect = "mysql"
	DialectSQLite   TargetDialect = "sqlite"
)

type TypeRule struct {
	Source      domain.FieldType
	Target      string
	Declaration string
	Warning     string
}

func RulesFor(dialect TargetDialect) map[domain.FieldType]TypeRule {
	rules := map[domain.FieldType]TypeRule{
		domain.TypeString:    {Source: domain.TypeString, Target: "text", Declaration: "TEXT"},
		domain.TypeInteger:   {Source: domain.TypeInteger, Target: "integer", Declaration: "INTEGER"},
		domain.TypeDecimal:   {Source: domain.TypeDecimal, Target: "decimal", Declaration: "DECIMAL(18,4)"},
		domain.TypeBoolean:   {Source: domain.TypeBoolean, Target: "boolean", Declaration: "BOOLEAN"},
		domain.TypeTimestamp: {Source: domain.TypeTimestamp, Target: "timestamp", Declaration: "TIMESTAMP"},
		domain.TypeDate:      {Source: domain.TypeDate, Target: "date", Declaration: "DATE"},
	}
	if dialect == DialectMySQL {
		rules[domain.TypeString] = TypeRule{Source: domain.TypeString, Target: "varchar", Declaration: "VARCHAR(255)"}
		rules[domain.TypeBoolean] = TypeRule{Source: domain.TypeBoolean, Target: "tinyint", Declaration: "TINYINT(1)", Warning: "boolean represented as tinyint"}
	}
	if dialect == DialectSQLite {
		rules[domain.TypeDecimal] = TypeRule{Source: domain.TypeDecimal, Target: "numeric", Declaration: "NUMERIC"}
		rules[domain.TypeTimestamp] = TypeRule{Source: domain.TypeTimestamp, Target: "text", Declaration: "TEXT", Warning: "timestamp represented as text"}
	}
	return rules
}

func DialectFor(name string) (TargetDialect, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "postgres", "postgresql":
		return DialectPostgres, nil
	case "mysql", "mariadb":
		return DialectMySQL, nil
	case "sqlite", "sqlite3":
		return DialectSQLite, nil
	default:
		return "", fmt.Errorf("unsupported target dialect %q", name)
	}
}

func QuoteIdentifier(name string, dialect TargetDialect) string {
	if dialect == DialectMySQL {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func NormalizeName(name string) string {
	clean := strings.TrimSpace(strings.ToLower(name))
	clean = strings.ReplaceAll(clean, "-", "_")
	clean = strings.ReplaceAll(clean, " ", "_")
	return clean
}
