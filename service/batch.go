package service

import (
	"context"
	"fmt"
	"sync"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

type BatchOutcome struct {
	Job     domain.BatchJob
	Records []domain.ConversionRecord
	Errors  []error
}

func (s *ConversionService) RunBatch(ctx context.Context, name string, requests []domain.ConversionRequest, workers int) (BatchOutcome, error) {
	if workers < 1 {
		return BatchOutcome{}, fmt.Errorf("workers must be positive")
	}
	sequence, err := s.repo.NextSequence()
	if err != nil {
		return BatchOutcome{}, err
	}
	job := domain.BatchJob{ID: storage.SequenceID("batch", sequence), Name: name, Status: "running", CreatedAt: fmt.Sprintf("sequence-%d", sequence)}
	for _, request := range requests {
		job.RequestKeys = append(job.RequestKeys, request.IdempotencyKey)
	}
	if err := s.repo.SaveBatch(job); err != nil {
		return BatchOutcome{}, err
	}
	type item struct {
		index   int
		request domain.ConversionRequest
	}
	items := make(chan item)
	results := make(chan struct {
		index  int
		record domain.ConversionRecord
		err    error
	}, len(requests))
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for next := range items {
				record, convertErr := s.Convert(ctx, next.request)
				results <- struct {
					index  int
					record domain.ConversionRecord
					err    error
				}{index: next.index, record: record, err: convertErr}
			}
		}()
	}
	go func() {
		defer close(results)
		for index, request := range requests {
			select {
			case items <- item{index: index, request: request}:
			case <-ctx.Done():
				return
			}
		}
		close(items)
		group.Wait()
	}()
	ordered := make([]domain.ConversionRecord, len(requests))
	seen := make([]bool, len(requests))
	for result := range results {
		if result.err != nil {
			job.Failed++
		} else {
			job.Processed++
			ordered[result.index] = result.record
			seen[result.index] = true
		}
	}
	for index := range ordered {
		if !seen[index] {
			job.Failed++
		}
	}
	job.Status = "completed"
	if job.Failed > 0 {
		job.Status = "completed_with_errors"
	}
	if err := s.repo.SaveBatch(job); err != nil {
		return BatchOutcome{}, err
	}
	return BatchOutcome{Job: job, Records: ordered}, nil
}
