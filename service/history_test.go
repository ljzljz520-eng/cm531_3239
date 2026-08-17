package service

import (
	"context"
	"testing"
)

func TestHistoryDownload(t *testing.T) {
	svc, repo := testService(t)
	record, err := svc.Convert(context.Background(), testRequest("history-key"))
	if err != nil {
		t.Fatal(err)
	}
	history := NewHistoryService(repo)
	download, err := history.Download(record.ID, "text")
	if err != nil {
		t.Fatal(err)
	}
	if download.Payload == "" || download.RecordID != record.ID {
		t.Fatalf("download %#v", download)
	}
	stored, err := repo.DownloadByID(download.ID)
	if err != nil || stored.Payload != download.Payload {
		t.Fatalf("stored %#v %v", stored, err)
	}
}
