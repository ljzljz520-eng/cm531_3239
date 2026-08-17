package storage

import (
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"

	bolt "go.etcd.io/bbolt"
)

type Snapshot struct {
	Records      int            `json:"records"`
	Batches      int            `json:"batches"`
	History      int            `json:"history"`
	Downloads    int            `json:"downloads"`
	Entities     map[string]int `json:"entities"`
	LastSequence int64          `json:"last_sequence"`
}

func (r *Repository) Snapshot() (Snapshot, error) {
	result := Snapshot{Entities: map[string]int{}}
	err := r.store.db.View(func(tx *bolt.Tx) error {
		result.Records = tx.Bucket([]byte(bucketRecords)).Stats().KeyN
		result.Batches = tx.Bucket([]byte(bucketBatches)).Stats().KeyN
		result.History = tx.Bucket([]byte(bucketHistory)).Stats().KeyN
		result.Downloads = tx.Bucket([]byte(bucketDownloads)).Stats().KeyN
		for _, bucket := range []string{bucketWorkOrders, bucketEquipment, bucketInspections, bucketMaterials} {
			result.Entities[bucket] = tx.Bucket([]byte(bucket)).Stats().KeyN
		}
		value := tx.Bucket([]byte(bucketMeta)).Get([]byte(sequenceKey))
		if _, err := fmt.Sscan(string(value), &result.LastSequence); err != nil {
			return err
		}
		return nil
	})
	return result, err
}

func (r *Repository) ListBatches() ([]domain.BatchJob, error) {
	result := []domain.BatchJob{}
	err := scanBucket(r.store.db, bucketBatches, func(value []byte) error {
		var job domain.BatchJob
		if err := decode(value, &job); err != nil {
			return err
		}
		result = append(result, job)
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result, err
}

func (r *Repository) ListDownloads(recordID string) ([]domain.Download, error) {
	result := []domain.Download{}
	err := scanBucket(r.store.db, bucketDownloads, func(value []byte) error {
		var download domain.Download
		if err := decode(value, &download); err != nil {
			return err
		}
		if recordID == "" || download.RecordID == recordID {
			result = append(result, download)
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result, err
}

func (r *Repository) ListAllHistory() ([]domain.HistoryEntry, error) {
	result := []domain.HistoryEntry{}
	err := scanBucket(r.store.db, bucketHistory, func(value []byte) error {
		var entry domain.HistoryEntry
		if err := decode(value, &entry); err != nil {
			return err
		}
		result = append(result, entry)
		return nil
	})
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].OccurredAt != result[j].OccurredAt {
			return result[i].OccurredAt < result[j].OccurredAt
		}
		return result[i].ID < result[j].ID
	})
	return result, err
}

func scanBucket(db *bolt.DB, bucket string, visit func([]byte) error) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s missing", bucket)
		}
		return b.ForEach(func(_, value []byte) error { return visit(value) })
	})
}

func (r *Repository) WorkOrderByID(id string) (domain.WorkOrder, error) {
	var value domain.WorkOrder
	return value, r.store.read(bucketWorkOrders, id, &value)
}

func (r *Repository) EquipmentByID(id string) (domain.Equipment, error) {
	var value domain.Equipment
	return value, r.store.read(bucketEquipment, id, &value)
}

func (r *Repository) InspectionByID(id string) (domain.QualityInspection, error) {
	var value domain.QualityInspection
	return value, r.store.read(bucketInspections, id, &value)
}

func (r *Repository) MaterialByID(id string) (domain.Material, error) {
	var value domain.Material
	return value, r.store.read(bucketMaterials, id, &value)
}

func (r *Repository) ListWorkOrders() ([]domain.WorkOrder, error) {
	return listEntities[domain.WorkOrder](r.store.db, bucketWorkOrders, func(value domain.WorkOrder) string { return value.OrderNo })
}

func (r *Repository) ListEquipment() ([]domain.Equipment, error) {
	return listEntities[domain.Equipment](r.store.db, bucketEquipment, func(value domain.Equipment) string { return value.Code })
}

func (r *Repository) ListInspections() ([]domain.QualityInspection, error) {
	return listEntities[domain.QualityInspection](r.store.db, bucketInspections, func(value domain.QualityInspection) string { return value.ID })
}

func (r *Repository) ListMaterials() ([]domain.Material, error) {
	return listEntities[domain.Material](r.store.db, bucketMaterials, func(value domain.Material) string { return value.SKU })
}

func listEntities[T any](db *bolt.DB, bucket string, key func(T) string) ([]T, error) {
	result := []T{}
	err := scanBucket(db, bucket, func(data []byte) error {
		var value T
		if err := decode(data, &value); err != nil {
			return err
		}
		result = append(result, value)
		return nil
	})
	sort.SliceStable(result, func(i, j int) bool { return key(result[i]) < key(result[j]) })
	return result, err
}

func (r *Repository) SearchRecords(term string) ([]domain.ConversionRecord, error) {
	records, err := r.ListRecords()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(term))
	if needle == "" {
		return records, nil
	}
	result := []domain.ConversionRecord{}
	for _, record := range records {
		fields := []string{record.ID, record.IdempotencyKey, record.SourceName, record.TargetName, record.RequestedBy, record.Status}
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), needle) {
				result = append(result, record)
				break
			}
		}
	}
	return result, nil
}

func (r *Repository) DeleteDownload(id string) error {
	return r.store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDownloads))
		if bucket.Get([]byte(id)) == nil {
			return fmt.Errorf("download %s not found", id)
		}
		return bucket.Delete([]byte(id))
	})
}
