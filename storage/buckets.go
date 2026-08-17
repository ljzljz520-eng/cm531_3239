package storage

import (
	"encoding/json"
	"fmt"
)

const (
	bucketRecords     = "conversion_records"
	bucketBatches     = "batch_jobs"
	bucketHistory     = "history_entries"
	bucketDownloads   = "downloads"
	bucketWorkOrders  = "WorkOrder"
	bucketEquipment   = "Equipment"
	bucketInspections = "QualityInspection"
	bucketMaterials   = "Material"
	bucketMeta        = "meta"
	sequenceKey       = "sequence"
)

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode value: %w", err)
	}
	return data, nil
}

func decode(data []byte, target any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty value")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode value: %w", err)
	}
	return nil
}
