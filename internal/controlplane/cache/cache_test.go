package cache

import (
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	c := New(5*time.Minute, 10*time.Minute)

	// Test set and get
	c.Set("key1", "value1", 0)
	val, found := c.Get("key1")
	if !found {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	// Test get non-existent key
	_, found = c.Get("nonexistent")
	if found {
		t.Error("expected not to find nonexistent key")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(5*time.Minute, 10*time.Minute)

	c.Set("key1", "value1", 0)
	c.Delete("key1")

	_, found := c.Get("key1")
	if found {
		t.Error("expected key1 to be deleted")
	}
}

func TestCache_Flush(t *testing.T) {
	c := New(5*time.Minute, 10*time.Minute)

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Flush()

	_, found1 := c.Get("key1")
	_, found2 := c.Get("key2")

	if found1 || found2 {
		t.Error("expected all keys to be flushed")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(50*time.Millisecond, 10*time.Minute)

	// Set with short TTL
	c.Set("key1", "value1", 50*time.Millisecond)

	// Should exist immediately
	_, found := c.Get("key1")
	if !found {
		t.Error("expected to find key1 before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = c.Get("key1")
	if found {
		t.Error("expected key1 to be expired")
	}
}

func TestCache_ItemCount(t *testing.T) {
	c := New(5*time.Minute, 10*time.Minute)

	if c.ItemCount() != 0 {
		t.Errorf("expected 0 items, got %d", c.ItemCount())
	}

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	if c.ItemCount() != 2 {
		t.Errorf("expected 2 items, got %d", c.ItemCount())
	}
}
