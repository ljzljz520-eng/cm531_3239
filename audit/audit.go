package audit

import (
	"fmt"
	"sort"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

type Logger struct {
	repo *storage.Repository
}

func NewLogger(repo *storage.Repository) *Logger { return &Logger{repo: repo} }

func (l *Logger) RecordConversion(record domain.ConversionRecord) error {
	entry := domain.HistoryEntry{ID: storage.SequenceID("audit", record.Sequence), RecordID: record.ID, Action: "conversion_completed", Actor: record.RequestedBy, OccurredAt: record.CompletedAt, Detail: fmt.Sprintf("%d tables, %d indexes", record.Summary.Tables, record.Summary.Indexes)}
	return l.repo.SaveHistory(entry)
}

func (l *Logger) RecordPreview(recordID, actor string, sequence int64) error {
	entry := domain.HistoryEntry{ID: storage.SequenceID("preview", sequence), RecordID: recordID, Action: "preview_viewed", Actor: actor, OccurredAt: fmt.Sprintf("sequence-%d-preview", sequence), Detail: "DDL preview inspected"}
	return l.repo.SaveHistory(entry)
}

func (l *Logger) RecordDownload(download domain.Download, actor string) error {
	entry := domain.HistoryEntry{ID: download.ID + "-audit", RecordID: download.RecordID, Action: "history_downloaded", Actor: actor, OccurredAt: download.CreatedAt, Detail: download.Format}
	return l.repo.SaveHistory(entry)
}

func (l *Logger) Timeline(recordID string) ([]domain.HistoryEntry, error) {
	entries, err := l.repo.History(recordID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].OccurredAt < entries[j].OccurredAt })
	return entries, nil
}
