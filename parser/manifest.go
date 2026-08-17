package parser

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"example.com/mestransform/domain"
)

type Manifest struct {
	CatalogName string
	Tables      []ManifestTable
	Metadata    map[string]string
}

type ManifestTable struct {
	Name       string
	Comment    string
	Columns    []ManifestColumn
	PrimaryKey []string
	Indexes    []ManifestIndex
}

type ManifestColumn struct {
	Name     string
	Type     string
	Nullable bool
	Length   int
	Comment  string
	Default  string
}

type ManifestIndex struct {
	Name    string
	Columns []string
	Unique  bool
	Comment string
}

func ParseManifest(reader io.Reader) (Manifest, error) {
	scanner := bufio.NewScanner(reader)
	manifest := Manifest{Metadata: map[string]string{}}
	var current *ManifestTable
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		for index := range parts {
			parts[index] = strings.TrimSpace(parts[index])
		}
		switch strings.ToLower(parts[0]) {
		case "catalog":
			if len(parts) != 2 || parts[1] == "" {
				return Manifest{}, fmt.Errorf("line %d: catalog requires a name", lineNumber)
			}
			manifest.CatalogName = parts[1]
		case "meta":
			if len(parts) != 3 {
				return Manifest{}, fmt.Errorf("line %d: metadata requires key and value", lineNumber)
			}
			manifest.Metadata[parts[1]] = parts[2]
		case "table":
			if len(parts) < 2 || parts[1] == "" {
				return Manifest{}, fmt.Errorf("line %d: table requires a name", lineNumber)
			}
			table := ManifestTable{Name: parts[1]}
			if len(parts) > 2 {
				table.Comment = parts[2]
			}
			manifest.Tables = append(manifest.Tables, table)
			current = &manifest.Tables[len(manifest.Tables)-1]
		case "column":
			if current == nil {
				return Manifest{}, fmt.Errorf("line %d: column appears before table", lineNumber)
			}
			column, err := parseManifestColumn(parts)
			if err != nil {
				return Manifest{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			current.Columns = append(current.Columns, column)
		case "primary":
			if current == nil || len(parts) < 2 {
				return Manifest{}, fmt.Errorf("line %d: invalid primary key", lineNumber)
			}
			current.PrimaryKey = append(current.PrimaryKey, splitNames(parts[1])...)
		case "index", "unique":
			if current == nil || len(parts) < 3 {
				return Manifest{}, fmt.Errorf("line %d: invalid index", lineNumber)
			}
			index := ManifestIndex{Name: parts[1], Columns: splitNames(parts[2]), Unique: strings.EqualFold(parts[0], "unique")}
			if len(parts) > 3 {
				index.Comment = parts[3]
			}
			current.Indexes = append(current.Indexes, index)
		default:
			return Manifest{}, fmt.Errorf("line %d: unsupported manifest directive %s", lineNumber, parts[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, err
	}
	if manifest.CatalogName == "" {
		return Manifest{}, fmt.Errorf("manifest catalog is missing")
	}
	if len(manifest.Tables) == 0 {
		return Manifest{}, fmt.Errorf("manifest contains no tables")
	}
	return manifest, nil
}

func parseManifestColumn(parts []string) (ManifestColumn, error) {
	if len(parts) < 3 || parts[1] == "" || parts[2] == "" {
		return ManifestColumn{}, fmt.Errorf("column requires name and type")
	}
	column := ManifestColumn{Name: parts[1], Type: parts[2], Nullable: true}
	if len(parts) > 3 {
		nullability := strings.ToLower(parts[3])
		if nullability == "required" || nullability == "not-null" {
			column.Nullable = false
		} else if nullability != "nullable" && nullability != "" {
			return ManifestColumn{}, fmt.Errorf("invalid nullability %s", parts[3])
		}
	}
	if len(parts) > 4 {
		if _, err := fmt.Sscan(parts[4], &column.Length); err != nil {
			return ManifestColumn{}, fmt.Errorf("invalid length %s", parts[4])
		}
	}
	if len(parts) > 5 {
		column.Comment = parts[5]
	}
	if len(parts) > 6 {
		column.Default = parts[6]
	}
	return column, nil
}

func splitNames(value string) []string {
	result := []string{}
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

func (m Manifest) Catalog() (domain.SourceCatalog, error) {
	catalog := domain.SourceCatalog{Name: m.CatalogName, Collected: m.Metadata["collected"]}
	for _, manifestTable := range m.Tables {
		table := domain.TableSpec{Name: manifestTable.Name, Comment: manifestTable.Comment, PrimaryKey: append([]string(nil), manifestTable.PrimaryKey...)}
		for _, manifestColumn := range manifestTable.Columns {
			fieldType, err := fieldTypeFor(manifestColumn.Type)
			if err != nil {
				return domain.SourceCatalog{}, fmt.Errorf("%s.%s: %w", manifestTable.Name, manifestColumn.Name, err)
			}
			table.Columns = append(table.Columns, domain.ColumnSpec{Name: manifestColumn.Name, Type: fieldType, Nullable: manifestColumn.Nullable, Length: manifestColumn.Length, Comment: manifestColumn.Comment, Default: manifestColumn.Default})
		}
		for _, manifestIndex := range manifestTable.Indexes {
			table.Indexes = append(table.Indexes, domain.IndexSpec{Name: manifestIndex.Name, Columns: append([]string(nil), manifestIndex.Columns...), Unique: manifestIndex.Unique, Comment: manifestIndex.Comment})
		}
		catalog.Tables = append(catalog.Tables, table)
	}
	sort.Slice(catalog.Tables, func(i, j int) bool { return catalog.Tables[i].Name < catalog.Tables[j].Name })
	if err := catalog.Validate(); err != nil {
		return domain.SourceCatalog{}, err
	}
	return catalog, nil
}

func fieldTypeFor(value string) (domain.FieldType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "string", "text", "varchar":
		return domain.TypeString, nil
	case "integer", "int", "bigint":
		return domain.TypeInteger, nil
	case "decimal", "numeric", "money":
		return domain.TypeDecimal, nil
	case "boolean", "bool":
		return domain.TypeBoolean, nil
	case "timestamp", "datetime":
		return domain.TypeTimestamp, nil
	case "date":
		return domain.TypeDate, nil
	default:
		return "", fmt.Errorf("unknown field type %s", value)
	}
}
