package service

import (
	"testing"
)

func TestUserAvatarService_ValidateFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		size        int64
		expectError bool
	}{
		{name: "valid jpeg", filename: "photo.jpg", contentType: "image/jpeg", size: 100000, expectError: false},
		{name: "valid png", filename: "avatar.png", contentType: "image/png", size: 500000, expectError: false},
		{name: "valid gif", filename: "anim.gif", contentType: "image/gif", size: 100000, expectError: false},
		{name: "valid webp", filename: "pic.webp", contentType: "image/webp", size: 200000, expectError: false},
		{name: "valid uppercase extension", filename: "photo.JPG", contentType: "image/jpeg", size: 100000, expectError: false},
		{name: "invalid content type", filename: "photo.jpg", contentType: "application/pdf", size: 100000, expectError: true},
		{name: "invalid extension", filename: "photo.bmp", contentType: "image/bmp", size: 100000, expectError: true},
		{name: "size too large", filename: "photo.jpg", contentType: "image/jpeg", size: AvatarMaxSize + 1, expectError: true},
		{name: "exact max size", filename: "photo.jpg", contentType: "image/jpeg", size: AvatarMaxSize, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &userAvatarService{}
			err := svc.ValidateFile(tt.filename, tt.contentType, tt.size)
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestUserAvatarConstants(t *testing.T) {
	if AvatarPrefix != "avatars/" {
		t.Errorf("AvatarPrefix = %q, want %q", AvatarPrefix, "avatars/")
	}
	if AvatarMaxSize != 2*1024*1024 {
		t.Errorf("AvatarMaxSize = %d, want %d", AvatarMaxSize, 2*1024*1024)
	}
}

func TestAllowedAvatarMimeTypes(t *testing.T) {
	expected := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	for mime := range expected {
		if !allowedAvatarMimeTypes[mime] {
			t.Errorf("expected %q to be allowed", mime)
		}
	}
	if len(allowedAvatarMimeTypes) != len(expected) {
		t.Errorf("expected %d allowed types, got %d", len(expected), len(allowedAvatarMimeTypes))
	}
}

func TestAllowedAvatarExtensions(t *testing.T) {
	expected := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	for ext := range expected {
		if !allowedAvatarExtensions[ext] {
			t.Errorf("expected %q to be allowed", ext)
		}
	}
	if len(allowedAvatarExtensions) != len(expected) {
		t.Errorf("expected %d allowed extensions, got %d", len(expected), len(allowedAvatarExtensions))
	}
}
