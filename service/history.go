package service

import (
	"fmt"
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

type HistoryService struct{ repo *storage.Repository }

func NewHistoryService(repo *storage.Repository) *HistoryService { return &HistoryService{repo: repo} }

func (s *HistoryService) Timeline(recordID string) ([]domain.HistoryEntry, error) {
	return s.repo.History(recordID)
}

func (s *HistoryService) Download(recordID, format string) (domain.Download, error) {
	record, err := s.repo.RecordByID(recordID)
	if err != nil {
		return domain.Download{}, err
	}
	if strings.TrimSpace(format) == "" {
		return domain.Download{}, fmt.Errorf("download format is required")
	}
	payload := "record=" + record.ID + "\nstatus=" + record.Status + "\ntables=" + fmt.Sprint(record.Summary.Tables) + "\n"
	sequence := record.Sequence
	download := domain.Download{ID: storage.SequenceID("download", sequence), RecordID: recordID, Format: format, Payload: payload, CreatedAt: fmt.Sprintf("sequence-%d-download", sequence)}
	if err := s.repo.SaveDownload(download); err != nil {
		return domain.Download{}, err
	}
	return download, nil
}

func (s *HistoryService) ReopenCheck(recordID string) (bool, error) {
	record, err := s.repo.RecordByID(recordID)
	if err != nil {
		return false, err
	}
	return record.Status == "completed" && record.ID == recordID, nil
}
