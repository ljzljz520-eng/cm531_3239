package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"example.com/mestransform/domain"
)

type RowSet struct {
	WorkOrders  []domain.WorkOrder
	Equipment   []domain.Equipment
	Inspections []domain.QualityInspection
	Materials   []domain.Material
}

func ParseRows(reader io.Reader, entity string) (RowSet, error) {
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return RowSet{}, err
	}
	if len(rows) == 0 {
		return RowSet{}, fmt.Errorf("empty csv")
	}
	headers := make(map[string]int)
	for i, header := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(header))] = i
	}
	set := RowSet{}
	for _, row := range rows[1:] {
		switch strings.ToLower(entity) {
		case "work_order":
			item, parseErr := parseWorkOrder(row, headers)
			if parseErr != nil {
				return RowSet{}, parseErr
			}
			set.WorkOrders = append(set.WorkOrders, item)
		case "equipment":
			item, parseErr := parseEquipment(row, headers)
			if parseErr != nil {
				return RowSet{}, parseErr
			}
			set.Equipment = append(set.Equipment, item)
		case "quality_inspection":
			item, parseErr := parseInspection(row, headers)
			if parseErr != nil {
				return RowSet{}, parseErr
			}
			set.Inspections = append(set.Inspections, item)
		case "material":
			item, parseErr := parseMaterial(row, headers)
			if parseErr != nil {
				return RowSet{}, parseErr
			}
			set.Materials = append(set.Materials, item)
		default:
			return RowSet{}, fmt.Errorf("unsupported entity %s", entity)
		}
	}
	return set, nil
}

func value(row []string, headers map[string]int, key string) string {
	index, ok := headers[key]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseWorkOrder(row []string, h map[string]int) (domain.WorkOrder, error) {
	quantity, err := strconv.Atoi(value(row, h, "quantity"))
	if err != nil {
		return domain.WorkOrder{}, fmt.Errorf("quantity: %w", err)
	}
	return domain.WorkOrder{ID: value(row, h, "id"), OrderNo: value(row, h, "order_no"), Product: value(row, h, "product"), Quantity: quantity, Status: value(row, h, "status"), ScheduleAt: value(row, h, "schedule_at")}, nil
}

func parseEquipment(row []string, h map[string]int) (domain.Equipment, error) {
	capacity, err := strconv.Atoi(value(row, h, "capacity"))
	if err != nil {
		return domain.Equipment{}, fmt.Errorf("capacity: %w", err)
	}
	return domain.Equipment{ID: value(row, h, "id"), Code: value(row, h, "code"), Name: value(row, h, "name"), Line: value(row, h, "line"), State: value(row, h, "state"), Capacity: capacity}, nil
}

func parseInspection(row []string, h map[string]int) (domain.QualityInspection, error) {
	passed, err := strconv.ParseBool(value(row, h, "passed"))
	if err != nil {
		return domain.QualityInspection{}, fmt.Errorf("passed: %w", err)
	}
	return domain.QualityInspection{ID: value(row, h, "id"), OrderID: value(row, h, "order_id"), Metric: value(row, h, "metric"), Expected: value(row, h, "expected"), Actual: value(row, h, "actual"), Passed: passed}, nil
}

func parseMaterial(row []string, h map[string]int) (domain.Material, error) {
	onHand, err := strconv.Atoi(value(row, h, "on_hand"))
	if err != nil {
		return domain.Material{}, fmt.Errorf("on_hand: %w", err)
	}
	reorder, err := strconv.Atoi(value(row, h, "reorder_at"))
	if err != nil {
		return domain.Material{}, fmt.Errorf("reorder_at: %w", err)
	}
	return domain.Material{ID: value(row, h, "id"), SKU: value(row, h, "sku"), Name: value(row, h, "name"), Unit: value(row, h, "unit"), OnHand: onHand, ReorderAt: reorder}, nil
}
