package asset

import (
	"testing"
)

func TestAsset_GetSizeFormatted(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{name: "zero bytes", size: 0, expected: "0 B"},
		{name: "bytes", size: 500, expected: "500 B"},
		{name: "exact KB", size: 1024, expected: "1 KB"},
		{name: "fractional KB", size: 1536, expected: "1.5 KB"},
		{name: "exact MB", size: 1048576, expected: "1 MB"},
		{name: "fractional MB", size: 5242880, expected: "5 MB"},
		{name: "exact GB", size: 1073741824, expected: "1 GB"},
		{name: "fractional GB", size: 2147483648, expected: "2 GB"},
		{name: "large GB", size: 5368709120, expected: "5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Asset{Size: tt.size}
			result := a.GetSizeFormatted()
			if result != tt.expected {
				t.Errorf("GetSizeFormatted() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAsset_IsImage(t *testing.T) {
	tests := []struct {
		category AssetCategory
		expected bool
	}{
		{CategoryImage, true},
		{CategoryVideo, false},
		{CategoryAudio, false},
		{CategoryDocument, false},
		{CategoryArchive, false},
		{CategoryOther, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			a := &Asset{Category: tt.category}
			if a.IsImage() != tt.expected {
				t.Errorf("IsImage() = %v, want %v for category %q", a.IsImage(), tt.expected, tt.category)
			}
		})
	}
}

func TestAsset_IsVideo(t *testing.T) {
	a := &Asset{Category: CategoryVideo}
	if !a.IsVideo() {
		t.Error("expected IsVideo() to be true for VIDEO category")
	}
	a2 := &Asset{Category: CategoryImage}
	if a2.IsVideo() {
		t.Error("expected IsVideo() to be false for IMAGE category")
	}
}

func TestAsset_IsAudio(t *testing.T) {
	a := &Asset{Category: CategoryAudio}
	if !a.IsAudio() {
		t.Error("expected IsAudio() to be true for AUDIO category")
	}
	a2 := &Asset{Category: CategoryDocument}
	if a2.IsAudio() {
		t.Error("expected IsAudio() to be false for DOCUMENT category")
	}
}

func TestAsset_IsDocument(t *testing.T) {
	a := &Asset{Category: CategoryDocument}
	if !a.IsDocument() {
		t.Error("expected IsDocument() to be true for DOCUMENT category")
	}
	a2 := &Asset{Category: CategoryArchive}
	if a2.IsDocument() {
		t.Error("expected IsDocument() to be false for ARCHIVE category")
	}
}

func TestAssetCategory_Constants(t *testing.T) {
	categories := map[AssetCategory]bool{
		CategoryImage:    true,
		CategoryVideo:    true,
		CategoryAudio:    true,
		CategoryDocument: true,
		CategoryArchive:  true,
		CategoryOther:    true,
	}
	for cat := range categories {
		if string(cat) == "" {
			t.Error("category should not be empty string")
		}
	}
}

func TestAssetStatus_Constants(t *testing.T) {
	statuses := map[AssetStatus]bool{
		StatusPending:  true,
		StatusActive:   true,
		StatusArchived: true,
		StatusDeleted:  true,
	}
	for st := range statuses {
		if string(st) == "" {
			t.Error("status should not be empty string")
		}
	}
}

func TestStorageType_Constants(t *testing.T) {
	types := map[StorageType]bool{
		StorageLocal: true,
		StorageS3:    true,
		StorageOSS:   true,
		StorageCOS:   true,
	}
	for st := range types {
		if string(st) == "" {
			t.Error("storage type should not be empty string")
		}
	}
}
