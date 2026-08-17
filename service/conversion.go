package service

import (
	"context"
	"fmt"
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
	"example.com/mestransform/storage"
)

type ConversionService struct {
	repo   *storage.Repository
	mapper mapping.Mapper
}

func NewConversionService(repo *storage.Repository, dialect mapping.TargetDialect) *ConversionService {
	return &ConversionService{repo: repo, mapper: mapping.NewMapper(dialect)}
}

func (s *ConversionService) Preview(ctx context.Context, request domain.ConversionRequest) (domain.Preview, error) {
	if err := ctx.Err(); err != nil {
		return domain.Preview{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.Preview{}, err
	}
	preview, err := s.mapper.Preview(request.Catalog)
	if err != nil {
		return domain.Preview{}, err
	}
	preview.RequestKey = request.IdempotencyKey
	preview.Statements = append(preview.Statements, s.mapper.IndexStatements(request.Catalog)...)
	preview.Summary.Indexes = len(preview.Statements) - preview.Summary.Tables
	return preview, nil
}

func (s *ConversionService) Convert(ctx context.Context, request domain.ConversionRequest) (domain.ConversionRecord, error) {
	if err := ctx.Err(); err != nil {
		return domain.ConversionRecord{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.ConversionRecord{}, err
	}
	// Idempotency: a repeated request with the same key returns the stored
	// result without re-executing the conversion flow. Dry runs are pure
	// previews and are not persisted, so they are not deduplicated.
	if !request.DryRun {
		if existing, ok, err := s.repo.RecordByKey(request.IdempotencyKey); err != nil {
			return domain.ConversionRecord{}, err
		} else if ok {
			return existing, nil
		}
	}
	preview, err := s.Preview(ctx, request)
	if err != nil {
		return domain.ConversionRecord{}, err
	}
	sequence, err := s.repo.NextSequence()
	if err != nil {
		return domain.ConversionRecord{}, err
	}
	record := domain.ConversionRecord{
		ID: storage.SequenceID("conversion", sequence), IdempotencyKey: request.IdempotencyKey,
		SourceName: request.SourceName, TargetName: request.TargetName, RequestedBy: request.RequestedBy,
		Status: "completed", Statements: preview.Statements, Summary: preview.Summary,
		CreatedAt: fmt.Sprintf("sequence-%d", sequence), CompletedAt: fmt.Sprintf("sequence-%d-complete", sequence), Sequence: sequence,
	}
	record.Summary.PersistedOrders = len(request.WorkOrders)
	record.Summary.PersistedAssets = len(request.Equipment)
	record.Summary.PersistedChecks = len(request.Inspections)
	record.Summary.PersistedStock = len(request.Materials)
	if request.DryRun {
		record.Status = "preview"
		return record, nil
	}
	stored, created, err := s.repo.SaveRecordIfAbsent(record)
	if err != nil {
		return domain.ConversionRecord{}, err
	}
	if !created {
		// A concurrent submission with this key completed first; reuse its result.
		return stored, nil
	}
	if err := s.repo.SaveEntities(request); err != nil {
		return domain.ConversionRecord{}, err
	}
	entry := domain.HistoryEntry{ID: storage.SequenceID("history", sequence), RecordID: record.ID, Action: "execute", Actor: request.RequestedBy, OccurredAt: record.CompletedAt, Detail: strings.Join([]string{request.SourceName, request.TargetName, record.Status}, " -> ")}
	if err := s.repo.SaveHistory(entry); err != nil {
		return domain.ConversionRecord{}, err
	}
	return record, nil
}

func (s *ConversionService) FindByIdempotencyKey(key string) (domain.ConversionRecord, bool, error) {
	return s.repo.RecordByKey(key)
}

func (s *ConversionService) AggregateSummary() (domain.MappingSummary, error) {
	records, err := s.repo.ListRecords()
	if err != nil {
		return domain.MappingSummary{}, err
	}
	var total domain.MappingSummary
	for _, record := range records {
		total.Tables += record.Summary.Tables
		total.Columns += record.Summary.Columns
		total.Indexes += record.Summary.Indexes
		total.Warnings += record.Summary.Warnings
		total.PersistedOrders += record.Summary.PersistedOrders
		total.PersistedAssets += record.Summary.PersistedAssets
		total.PersistedChecks += record.Summary.PersistedChecks
		total.PersistedStock += record.Summary.PersistedStock
	}
	return total, nil
}
