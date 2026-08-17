package mapping

import (
	"fmt"
	"strings"

	"example.com/mestransform/domain"
)

type CommentPolicy struct {
	IncludeTableComments  bool
	IncludeColumnComments bool
	IncludeIndexComments  bool
	Prefix                string
}

func DefaultCommentPolicy() CommentPolicy {
	return CommentPolicy{IncludeTableComments: true, IncludeColumnComments: true, IncludeIndexComments: true, Prefix: "MES"}
}

func TableComment(table domain.TableSpec, policy CommentPolicy) string {
	if !policy.IncludeTableComments {
		return ""
	}
	text := strings.TrimSpace(table.Comment)
	if text == "" {
		text = "MES table " + table.Name
	}
	if policy.Prefix != "" {
		return policy.Prefix + ": " + text
	}
	return text
}

func ColumnComment(column domain.ColumnSpec, policy CommentPolicy) string {
	if !policy.IncludeColumnComments {
		return ""
	}
	if strings.TrimSpace(column.Comment) == "" {
		return fmt.Sprintf("MES column %s", column.Name)
	}
	return strings.TrimSpace(column.Comment)
}

func IndexComment(index domain.IndexSpec, policy CommentPolicy) string {
	if !policy.IncludeIndexComments {
		return ""
	}
	if strings.TrimSpace(index.Comment) == "" {
		return "MES index " + index.Name
	}
	return strings.TrimSpace(index.Comment)
}

func RenderComment(comment, dialect string) string {
	if comment == "" {
		return ""
	}
	if strings.EqualFold(dialect, "mysql") {
		return " COMMENT '" + strings.ReplaceAll(comment, "'", "''") + "'"
	}
	return " /* " + strings.ReplaceAll(comment, "*/", "* /") + " */"
}
