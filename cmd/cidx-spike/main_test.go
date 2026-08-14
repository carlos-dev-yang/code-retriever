package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProductionIndexPathRejectsDirectAndSymlinkedPaths(t *testing.T) {
	root := t.TempDir()
	productionDirectory := filepath.Join(root, "project", ".cidx")
	if err := os.MkdirAll(productionDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	productionPath := filepath.Join(productionDirectory, "index.db")
	if err := os.WriteFile(productionPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !isProductionIndexPath(productionPath) || !isProductionIndexPath(filepath.Join(".cidx", "index.db")) {
		t.Fatal("direct production-shaped path was accepted")
	}
	fileLink := filepath.Join(root, "production-link.db")
	if err := os.Symlink(productionPath, fileLink); err != nil {
		t.Fatal(err)
	}
	if !isProductionIndexPath(fileLink) {
		t.Fatal("symlinked production file was accepted")
	}
	directoryLink := filepath.Join(root, "cidx-link")
	if err := os.Symlink(productionDirectory, directoryLink); err != nil {
		t.Fatal(err)
	}
	if !isProductionIndexPath(filepath.Join(directoryLink, "index.db")) {
		t.Fatal("path through symlinked production directory was accepted")
	}
	if isProductionIndexPath(filepath.Join(root, "spikes", "index.db")) {
		t.Fatal("non-production spike path was rejected")
	}
}
