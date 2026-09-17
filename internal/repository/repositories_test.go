package repository

import (
	"os"
	"testing"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
)

func setupTestDB(t *testing.T) (*Repository, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "test-stream-db-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	database, err := db.InitDB(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init test DB: %v", err)
	}

	repo := NewRepository(database)
	cleanup := func() {
		database.Close()
		os.RemoveAll(tempDir)
	}
	return repo, cleanup
}

func TestUserRepository(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := repo.CountUsers()
	if err != nil || count != 0 {
		t.Fatalf("Expected 0 users initially, got %d (err: %v)", count, err)
	}

	user, err := repo.CreateUser("admin", "$2a$10$dummyhash")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", user.Username)
	}

	fetched, err := repo.GetUserByUsername("admin")
	if err != nil || fetched.ID != user.ID {
		t.Errorf("Failed to retrieve created user: %v", err)
	}
}

func TestCategoriesAndStreamRepository(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	cat, err := repo.CreateCategory("Sports", "sports", 1)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	stream := &models.Stream{
		Name:              "Test ESPN",
		Slug:              "test-espn",
		MediaURL:          "udp://@239.255.0.1:1234",
		TVGId:             "espn.us",
		ProgramID:         "101",
		LocalAddr:         "192.168.1.55",
		Mode:              "ondemand",
		IdleTimeoutSec:    120,
		AutoRecover:       true,
		RecoverTimeoutSec: 25,
		Enabled:           true,
		CategoryIDs:       []string{cat.ID},
	}

	created, err := repo.CreateStream(stream)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	if created.ID == "" {
		t.Errorf("Expected stream ID to be generated")
	}

	fetched, err := repo.GetStreamByID(created.ID)
	if err != nil {
		t.Fatalf("Failed to fetch stream: %v", err)
	}

	if fetched.Name != "Test ESPN" || !fetched.AutoRecover || fetched.RecoverTimeoutSec != 25 || fetched.LocalAddr != "192.168.1.55" || fetched.ProgramID != "101" {
		t.Errorf("Stream data mismatch: %+v", fetched)
	}

	if len(fetched.Categories) != 1 || fetched.Categories[0].Name != "Sports" {
		t.Errorf("Expected category 'Sports' to be mapped, got %+v", fetched.Categories)
	}

	// Test updating LocalAddr
	fetched.LocalAddr = "10.0.0.42"
	if err := repo.UpdateStream(fetched); err != nil {
		t.Fatalf("Failed to update stream: %v", err)
	}

	updated, err := repo.GetStreamBySlug(fetched.Slug)
	if err != nil {
		t.Fatalf("Failed to fetch updated stream by slug: %v", err)
	}
	if updated.LocalAddr != "10.0.0.42" {
		t.Errorf("Expected updated LocalAddr '10.0.0.42', got '%s'", updated.LocalAddr)
	}
}
