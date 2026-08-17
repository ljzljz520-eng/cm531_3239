package domain

import (
	"fmt"
	"strings"
)

type WorkOrder struct {
	ID         string `json:"id"`
	OrderNo    string `json:"order_no"`
	Product    string `json:"product"`
	Quantity   int    `json:"quantity"`
	Status     string `json:"status"`
	ScheduleAt string `json:"schedule_at"`
}

type Equipment struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Line     string `json:"line"`
	State    string `json:"state"`
	Capacity int    `json:"capacity"`
}

type QualityInspection struct {
	ID       string `json:"id"`
	OrderID  string `json:"order_id"`
	Metric   string `json:"metric"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

type Material struct {
	ID        string `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	OnHand    int    `json:"on_hand"`
	ReorderAt int    `json:"reorder_at"`
}

type ConversionRequest struct {
	IdempotencyKey string              `json:"idempotency_key"`
	SourceName     string              `json:"source_name"`
	TargetName     string              `json:"target_name"`
	RequestedBy    string              `json:"requested_by"`
	Catalog        SourceCatalog       `json:"catalog"`
	WorkOrders     []WorkOrder         `json:"work_orders"`
	Equipment      []Equipment         `json:"equipment"`
	Inspections    []QualityInspection `json:"inspections"`
	Materials      []Material          `json:"materials"`
	DryRun         bool                `json:"dry_run"`
}

func (r ConversionRequest) Validate() error {
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(r.SourceName) == "" || strings.TrimSpace(r.TargetName) == "" {
		return fmt.Errorf("source and target names are required")
	}
	if strings.EqualFold(r.SourceName, r.TargetName) {
		return fmt.Errorf("source and target must differ")
	}
	if err := r.Catalog.Validate(); err != nil {
		return err
	}
	for _, order := range r.WorkOrders {
		if order.ID == "" || order.OrderNo == "" || order.Quantity <= 0 {
			return fmt.Errorf("invalid work order %s", order.ID)
		}
	}
	for _, equipment := range r.Equipment {
		if equipment.ID == "" || equipment.Code == "" {
			return fmt.Errorf("invalid equipment %s", equipment.ID)
		}
	}
	for _, inspection := range r.Inspections {
		if inspection.ID == "" || inspection.OrderID == "" {
			return fmt.Errorf("invalid inspection %s", inspection.ID)
		}
	}
	for _, material := range r.Materials {
		if material.ID == "" || material.SKU == "" || material.OnHand < 0 {
			return fmt.Errorf("invalid material %s", material.ID)
		}
	}
	return nil
}

type ConversionRecord struct {
	ID             string         `json:"id"`
	IdempotencyKey string         `json:"idempotency_key"`
	SourceName     string         `json:"source_name"`
	TargetName     string         `json:"target_name"`
	RequestedBy    string         `json:"requested_by"`
	Status         string         `json:"status"`
	Statements     []DDLStatement `json:"statements"`
	Summary        MappingSummary `json:"summary"`
	CreatedAt      string         `json:"created_at"`
	CompletedAt    string         `json:"completed_at"`
	Sequence       int64          `json:"sequence"`
}

type BatchJob struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	RequestKeys []string `json:"request_keys"`
	Status      string   `json:"status"`
	Processed   int      `json:"processed"`
	Failed      int      `json:"failed"`
	CreatedAt   string   `json:"created_at"`
}

type HistoryEntry struct {
	ID         string `json:"id"`
	RecordID   string `json:"record_id"`
	Action     string `json:"action"`
	Actor      string `json:"actor"`
	OccurredAt string `json:"occurred_at"`
	Detail     string `json:"detail"`
}
