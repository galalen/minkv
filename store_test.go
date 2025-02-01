package minkv

import (
	"os"
	"testing"
)

func setupKV(t *testing.T) *Store {
	t.Helper()
	s, err := Open("test.db")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	return s
}

func cleanupKV(t *testing.T, s *Store) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}
	if err := os.Remove("test.db"); err != nil {
		t.Fatalf("failed to remove test.db: %v", err)
	}
}

func TestPutAndGet(t *testing.T) {
	store := setupKV(t)
	defer cleanupKV(t, store)

	key := []byte("key1")
	value := []byte("value1")

	if err := store.Put(key, value); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrievedValue, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrievedValue) != string(value) {
		t.Errorf("Expected value %s, got %s", value, retrievedValue)
	}
}

func TestUpdate(t *testing.T) {
	store := setupKV(t)
	defer cleanupKV(t, store)

	key := []byte("key1")
	value1 := []byte("value1")
	value2 := []byte("value2")

	if err := store.Put(key, value1); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// update value
	if err := store.Put(key, value2); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrievedValue, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrievedValue) != string(value2) {
		t.Errorf("Expected value %s, got %s", value2, retrievedValue)
	}
}

func TestDelete(t *testing.T) {
	store := setupKV(t)
	defer cleanupKV(t, store)

	key := []byte("key1")
	value := []byte("value1")

	if err := store.Put(key, value); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get(key)
	if err == nil {
		t.Error("Expected error for deleted key, got nil")
	}
}

func TestIterator(t *testing.T) {
	store := setupKV(t)
	defer cleanupKV(t, store)

	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	for key, value := range data {
		if err := store.Put([]byte(key), []byte(value)); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	it, err := store.Iterator()
	if err != nil {
		t.Fatalf("Iterator failed: %v", err)
	}

	found := make(map[string]string)
	for it.Next() {
		record, err := it.Record()
		if err != nil {
			t.Fatalf("Record failed: %v", err)
		}
		found[string(record.Key)] = string(record.Value)
	}

	for key, value := range data {
		if found[key] != value {
			t.Errorf("Expected %s for key %s, got %s", value, key, found[key])
		}
	}
}
