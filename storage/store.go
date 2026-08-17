package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

type Store struct {
	db   *bolt.DB
	path string
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("storage path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage directory: %w", err)
		}
	}
	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt store: %w", err)
	}
	store := &Store{db: db, path: path}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{bucketRecords, bucketBatches, bucketHistory, bucketDownloads, bucketWorkOrders, bucketEquipment, bucketInspections, bucketMaterials, bucketMeta} {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("create bucket %s: %w", name, err)
			}
		}
		meta := tx.Bucket([]byte(bucketMeta))
		if meta.Get([]byte(sequenceKey)) == nil {
			return meta.Put([]byte(sequenceKey), []byte("0"))
		}
		return nil
	})
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string { return s.path }

func (s *Store) nextSequence(tx *bolt.Tx) (int64, error) {
	meta := tx.Bucket([]byte(bucketMeta))
	value := meta.Get([]byte(sequenceKey))
	var current int64
	if _, err := fmt.Sscan(string(value), &current); err != nil {
		return 0, fmt.Errorf("read sequence: %w", err)
	}
	current++
	if err := meta.Put([]byte(sequenceKey), []byte(fmt.Sprintf("%d", current))); err != nil {
		return 0, err
	}
	return current, nil
}

func (s *Store) read(bucket, key string, target any) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s missing", bucket)
		}
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("%s %s not found", bucket, key)
		}
		return decode(data, target)
	})
}
