package service

import (
	"strings"
	"testing"
)

func TestFeatureImageHelpers(t *testing.T) {
	if !strings.Contains(newFeatureImageUploadID(9, 1), "feature_image_9_1_") {
		t.Fatal("upload id")
	}
	if got := featureImageUploadPath(9); got != "uploads/features/9" {
		t.Fatalf("path=%s", got)
	}

	if prependPublicURL("https://cdn", "") != "" {
		t.Fatal("empty path")
	}
	if prependPublicURL("", "rel") != "rel" {
		t.Fatal("empty base")
	}
	if prependPublicURL("https://cdn", "https://other/a") != "https://other/a" {
		t.Fatal("absolute")
	}
	if got := prependPublicURL("https://cdn/", "/img.jpg"); got != "https://cdn/img.jpg" {
		t.Fatalf("join=%s", got)
	}

	if err := validateFeatureImage(nil, "a.jpg", "image/jpeg"); err != ErrFeatureImageRequired {
		t.Fatalf("required: %v", err)
	}
	big := make([]byte, featureImageMaxSize+1)
	if err := validateFeatureImage(big, "a.jpg", "image/jpeg"); err != ErrInvalidFeatureImage {
		t.Fatalf("too large: %v", err)
	}
	if err := validateFeatureImage([]byte{1}, "a.gif", "image/gif"); err != ErrInvalidFeatureImage {
		t.Fatalf("bad type: %v", err)
	}
	if err := validateFeatureImage([]byte{1}, "a.gif", "image/jpeg"); err != ErrInvalidFeatureImage {
		t.Fatalf("bad name: %v", err)
	}
	if err := validateFeatureImage([]byte{1}, "a.jpg", "image/jpeg; charset=binary"); err != nil {
		t.Fatalf("ok jpeg: %v", err)
	}
	if !isAllowedFeatureImageContentType("image/png") || isAllowedFeatureImageContentType("image/webp") {
		t.Fatal("content types")
	}
	if !isAllowedFeatureImageFilename("x.BMP") || isAllowedFeatureImageFilename("x.txt") {
		t.Fatal("filenames")
	}
	if defaultFeatureImageFilename("image/png", 0) != "image_1.png" {
		t.Fatal("png name")
	}
	if defaultFeatureImageFilename("image/bmp", 1) != "image_2.bmp" {
		t.Fatal("bmp name")
	}
	if defaultFeatureImageFilename("image/jpeg", 2) != "image_3.jpg" {
		t.Fatal("jpg name")
	}
}
