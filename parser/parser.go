package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"example.com/mestransform/domain"
)

type InputFormat string

const (
	FormatJSON InputFormat = "json"
	FormatDDL  InputFormat = "ddl"
)

type Parser struct {
	Strict bool
}

func New(strict bool) Parser { return Parser{Strict: strict} }

func (p Parser) ParseJSON(data []byte) (domain.SourceCatalog, error) {
	var catalog domain.SourceCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return domain.SourceCatalog{}, fmt.Errorf("decode catalog: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return domain.SourceCatalog{}, err
	}
	return catalog, nil
}

func (p Parser) Parse(reader io.Reader, format InputFormat) (domain.SourceCatalog, error) {
	if format == FormatJSON {
		data, err := io.ReadAll(reader)
		if err != nil {
			return domain.SourceCatalog{}, err
		}
		return p.ParseJSON(data)
	}
	if format == FormatDDL {
		return p.parseDDL(reader)
	}
	return domain.SourceCatalog{}, fmt.Errorf("unsupported input format %s", format)
}

func (p Parser) parseDDL(reader io.Reader) (domain.SourceCatalog, error) {
	scanner := bufio.NewScanner(reader)
	catalog := domain.SourceCatalog{Name: "ddl-import"}
	var current *domain.TableSpec
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "CREATE TABLE") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line[len("CREATE TABLE"):], "IF NOT EXISTS"), "("))
			name = strings.Trim(name, " `\"")
			catalog.Tables = append(catalog.Tables, domain.TableSpec{Name: name})
			current = &catalog.Tables[len(catalog.Tables)-1]
			continue
		}
		if current == nil {
			if p.Strict {
				return domain.SourceCatalog{}, fmt.Errorf("statement before table")
			}
			continue
		}
		if strings.HasPrefix(upper, "PRIMARY KEY") {
			start := strings.Index(line, "(")
			end := strings.LastIndex(line, ")")
			if start >= 0 && end > start {
				current.PrimaryKey = append(current.PrimaryKey, strings.Trim(strings.TrimSpace(line[start+1:end]), " `\""))
			}
			continue
		}
		if strings.HasPrefix(upper, "CONSTRAINT") || strings.HasPrefix(upper, "CREATE INDEX") {
			continue
		}
		parts := strings.Fields(strings.TrimSuffix(line, ","))
		if len(parts) >= 2 {
			column := domain.ColumnSpec{Name: strings.Trim(parts[0], " `\""), Type: parseType(parts[1]), Nullable: true}
			if strings.Contains(upper, "NOT NULL") {
				column.Nullable = false
			}
			current.Columns = append(current.Columns, column)
		}
	}
	if err := scanner.Err(); err != nil {
		return domain.SourceCatalog{}, err
	}
	if err := catalog.Validate(); err != nil {
		return domain.SourceCatalog{}, err
	}
	return catalog, nil
}

func parseType(raw string) domain.FieldType {
	t := strings.ToLower(raw)
	switch {
	case strings.Contains(t, "int"):
		return domain.TypeInteger
	case strings.Contains(t, "decimal"), strings.Contains(t, "numeric"), strings.Contains(t, "real"):
		return domain.TypeDecimal
	case strings.Contains(t, "bool"):
		return domain.TypeBoolean
	case strings.Contains(t, "timestamp"), strings.Contains(t, "datetime"):
		return domain.TypeTimestamp
	case t == "date":
		return domain.TypeDate
	default:
		return domain.TypeString
	}
}
