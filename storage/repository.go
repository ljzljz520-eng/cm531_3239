package storage

import (
	"fmt"
	"sort"
	"strconv"

	"example.com/mestransform/domain"

	bolt "go.etcd.io/bbolt"
)

type Repository struct{ store *Store }

func NewRepository(store *Store) *Repository { return &Repository{store: store} }

func (r *Repository) SaveRecord(record domain.ConversionRecord) error {
	data, err := encode(record)
	if err != nil {
		return err
	}
	return r.store.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucketRecords)).Put([]byte(record.ID), data)
	})
}

func (r *Repository) RecordByID(id string) (domain.ConversionRecord, error) {
	var record domain.ConversionRecord
	return record, r.store.read(bucketRecords, id, &record)
}

func (r *Repository) RecordByKey(key string) (domain.ConversionRecord, bool, error) {
	var records []domain.ConversionRecord
	err := r.store.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucketRecords)).ForEach(func(_, value []byte) error {
			var record domain.ConversionRecord
			if err := decode(value, &record); err != nil {
				return err
			}
			if record.IdempotencyKey == key {
				records = append(records, record)
			}
			return nil
		})
	})
	if err != nil {
		return domain.ConversionRecord{}, false, err
	}
	if len(records) == 0 {
		return domain.ConversionRecord{}, false, nil
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records[0], true, nil
}

func (r *Repository) ListRecords() ([]domain.ConversionRecord, error) {
	result := []domain.ConversionRecord{}
	err := r.store.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucketRecords)).ForEach(func(_, value []byte) error {
			var record domain.ConversionRecord
			if err := decode(value, &record); err != nil {
				return err
			}
			result = append(result, record)
			return nil
		})
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, err
}

func (r *Repository) SaveBatch(job domain.BatchJob) error {
	data, err := encode(job)
	if err != nil {
		return err
	}
	return r.store.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte(bucketBatches)).Put([]byte(job.ID), data) })
}

func (r *Repository) BatchByID(id string) (domain.BatchJob, error) {
	var job domain.BatchJob
	return job, r.store.read(bucketBatches, id, &job)
}

func (r *Repository) SaveHistory(entry domain.HistoryEntry) error {
	data, err := encode(entry)
	if err != nil {
		return err
	}
	return r.store.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte(bucketHistory)).Put([]byte(entry.ID), data) })
}

func (r *Repository) History(recordID string) ([]domain.HistoryEntry, error) {
	result := []domain.HistoryEntry{}
	err := r.store.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucketHistory)).ForEach(func(_, value []byte) error {
			var entry domain.HistoryEntry
			if err := decode(value, &entry); err != nil {
				return err
			}
			if entry.RecordID == recordID {
				result = append(result, entry)
			}
			return nil
		})
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, err
}

func (r *Repository) SaveDownload(download domain.Download) error {
	data, err := encode(download)
	if err != nil {
		return err
	}
	return r.store.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte(bucketDownloads)).Put([]byte(download.ID), data) })
}

func (r *Repository) DownloadByID(id string) (domain.Download, error) {
	var download domain.Download
	return download, r.store.read(bucketDownloads, id, &download)
}

func (r *Repository) SaveEntities(request domain.ConversionRequest) error {
	return r.store.db.Update(func(tx *bolt.Tx) error {
		if err := putEntities(tx, bucketWorkOrders, request.WorkOrders, func(v domain.WorkOrder) string { return v.ID }); err != nil {
			return err
		}
		if err := putEntities(tx, bucketEquipment, request.Equipment, func(v domain.Equipment) string { return v.ID }); err != nil {
			return err
		}
		if err := putEntities(tx, bucketInspections, request.Inspections, func(v domain.QualityInspection) string { return v.ID }); err != nil {
			return err
		}
		return putEntities(tx, bucketMaterials, request.Materials, func(v domain.Material) string { return v.ID })
	})
}

func putEntities[T any](tx *bolt.Tx, bucket string, values []T, key func(T) string) error {
	b := tx.Bucket([]byte(bucket))
	for _, value := range values {
		data, err := encode(value)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(key(value)), data); err != nil {
			return fmt.Errorf("save %s: %w", bucket, err)
		}
	}
	return nil
}

func (r *Repository) EntityCounts() (map[string]int, error) {
	counts := map[string]int{}
	err := r.store.db.View(func(tx *bolt.Tx) error {
		for _, bucket := range []string{bucketWorkOrders, bucketEquipment, bucketInspections, bucketMaterials} {
			counts[bucket] = tx.Bucket([]byte(bucket)).Stats().KeyN
		}
		return nil
	})
	return counts, err
}

func (r *Repository) NextSequence() (int64, error) {
	var sequence int64
	err := r.store.db.Update(func(tx *bolt.Tx) error {
		var err error
		sequence, err = r.store.nextSequence(tx)
		return err
	})
	return sequence, err
}

func SequenceID(prefix string, sequence int64) string {
	return prefix + "-" + strconv.FormatInt(sequence, 10)
}
