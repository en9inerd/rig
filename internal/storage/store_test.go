package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "store.json")
}

func TestNew_NonExistentFile(t *testing.T) {
	s, err := New(tempPath(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Len("any") != 0 {
		t.Fatal("expected empty store")
	}
}

func TestNew_ExistingFile(t *testing.T) {
	path := tempPath(t)
	data := map[string]map[string]Entry{
		"b1": {"k1": {Value: "v1", SetAt: 100}},
	}
	raw, _ := json.Marshal(data)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	v, ok := s.Get("b1", "k1")
	if !ok || v != "v1" {
		t.Fatalf("Get = (%q, %v), want (\"v1\", true)", v, ok)
	}
}

func TestNew_CorruptFile(t *testing.T) {
	path := tempPath(t)
	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(path)
	if err == nil {
		t.Fatal("expected error for corrupt file")
	}
}

func TestNew_EmptyFile(t *testing.T) {
	path := tempPath(t)
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Len("any") != 0 {
		t.Fatal("expected empty store")
	}
}

func TestNew_CreatesParentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep")
	path := filepath.Join(dir, "store.json")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Set("b", "k", "v"); err != nil {
		t.Fatalf("Set after dir creation: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("store file not created: %v", err)
	}
}

func TestNew_PermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}

	path := tempPath(t)
	if err := os.WriteFile(path, []byte(`{"b":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	_, err := New(path)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestSetAndGet(t *testing.T) {
	s, _ := New(tempPath(t))

	if err := s.Set("b", "k", "v"); err != nil {
		t.Fatal(err)
	}

	v, ok := s.Get("b", "k")
	if !ok || v != "v" {
		t.Fatalf("Get = (%q, %v), want (\"v\", true)", v, ok)
	}
}

func TestGet_MissingBucket(t *testing.T) {
	s, _ := New(tempPath(t))

	v, ok := s.Get("missing", "k")
	if ok || v != "" {
		t.Fatalf("Get = (%q, %v), want (\"\", false)", v, ok)
	}
}

func TestGet_MissingKey(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k1", "v1")

	v, ok := s.Get("b", "missing")
	if ok || v != "" {
		t.Fatalf("Get = (%q, %v), want (\"\", false)", v, ok)
	}
}

func TestHas(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k", "v")

	if !s.Has("b", "k") {
		t.Fatal("Has = false, want true")
	}
	if s.Has("b", "missing") {
		t.Fatal("Has = true for missing key")
	}
	if s.Has("missing", "k") {
		t.Fatal("Has = true for missing bucket")
	}
}

func TestSetOverwrite(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k", "v1")
	_ = s.Set("b", "k", "v2")

	v, _ := s.Get("b", "k")
	if v != "v2" {
		t.Fatalf("Get = %q, want \"v2\"", v)
	}
}

func TestSetBatch(t *testing.T) {
	s, _ := New(tempPath(t))

	pairs := map[string]string{"a": "1", "b": "2", "c": "3"}
	if err := s.SetBatch("bucket", pairs); err != nil {
		t.Fatal(err)
	}

	if s.Len("bucket") != 3 {
		t.Fatalf("Len = %d, want 3", s.Len("bucket"))
	}
	for k, v := range pairs {
		got, ok := s.Get("bucket", k)
		if !ok || got != v {
			t.Fatalf("Get(%q) = (%q, %v), want (%q, true)", k, got, ok, v)
		}
	}
}

func TestDelete(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k", "v")

	if err := s.Delete("b", "k"); err != nil {
		t.Fatal(err)
	}
	if s.Has("b", "k") {
		t.Fatal("key still exists after delete")
	}
}

func TestDelete_MissingBucket(t *testing.T) {
	s, _ := New(tempPath(t))
	if err := s.Delete("missing", "k"); err != nil {
		t.Fatalf("Delete missing bucket: %v", err)
	}
}

func TestDelete_MissingKey(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k", "v")

	if err := s.Delete("b", "missing"); err != nil {
		t.Fatalf("Delete missing key: %v", err)
	}
	if !s.Has("b", "k") {
		t.Fatal("existing key was removed")
	}
}

func TestForEach(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.SetBatch("b", map[string]string{"a": "1", "b": "2"})

	got := make(map[string]string)
	err := s.ForEach("b", func(k, v string) error {
		got[k] = v
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("ForEach collected %v", got)
	}
}

func TestForEach_EmptyBucket(t *testing.T) {
	s, _ := New(tempPath(t))

	called := false
	_ = s.ForEach("empty", func(_, _ string) error {
		called = true
		return nil
	})
	if called {
		t.Fatal("fn called on empty bucket")
	}
}

func TestKeys(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.SetBatch("b", map[string]string{"x": "1", "y": "2"})

	keys := s.Keys("b")
	if len(keys) != 2 {
		t.Fatalf("Keys = %v, want 2 keys", keys)
	}
	m := map[string]bool{}
	for _, k := range keys {
		m[k] = true
	}
	if !m["x"] || !m["y"] {
		t.Fatalf("Keys = %v, want [x, y]", keys)
	}
}

func TestKeys_EmptyBucket(t *testing.T) {
	s, _ := New(tempPath(t))
	keys := s.Keys("empty")
	if len(keys) != 0 {
		t.Fatalf("Keys = %v, want empty", keys)
	}
}

func TestLen(t *testing.T) {
	s, _ := New(tempPath(t))
	if s.Len("b") != 0 {
		t.Fatal("Len of new bucket != 0")
	}

	_ = s.Set("b", "k1", "v")
	_ = s.Set("b", "k2", "v")
	if s.Len("b") != 2 {
		t.Fatalf("Len = %d, want 2", s.Len("b"))
	}
}

func TestPrune(t *testing.T) {
	s, _ := New(tempPath(t))

	s.mu.Lock()
	b := s.bucket("b")
	b["old"] = Entry{Value: "x", SetAt: 1000}
	b["new"] = Entry{Value: "y", SetAt: time.Now().Unix()}
	_ = s.flush()
	s.mu.Unlock()

	removed, err := s.Prune("b", time.Unix(2000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("Prune removed %d, want 1", removed)
	}
	if s.Has("b", "old") {
		t.Fatal("old entry still exists")
	}
	if !s.Has("b", "new") {
		t.Fatal("new entry was pruned")
	}
}

func TestPrune_EmptyBucket(t *testing.T) {
	s, _ := New(tempPath(t))
	removed, err := s.Prune("empty", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("Prune removed %d, want 0", removed)
	}
}

func TestPrune_NothingOld(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b", "k", "v")

	removed, err := s.Prune("b", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("Prune removed %d, want 0", removed)
	}
}

func TestBucketIsolation(t *testing.T) {
	s, _ := New(tempPath(t))
	_ = s.Set("b1", "k", "v1")
	_ = s.Set("b2", "k", "v2")

	v1, _ := s.Get("b1", "k")
	v2, _ := s.Get("b2", "k")

	if v1 != "v1" || v2 != "v2" {
		t.Fatalf("bucket isolation broken: b1=%q, b2=%q", v1, v2)
	}
}

func TestPersistence(t *testing.T) {
	path := tempPath(t)

	s1, _ := New(path)
	_ = s1.Set("b", "k", "v")
	_ = s1.SetBatch("b2", map[string]string{"a": "1"})

	s2, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := s2.Get("b", "k")
	if !ok || v != "v" {
		t.Fatalf("persistence: Get = (%q, %v)", v, ok)
	}
	v2, ok := s2.Get("b2", "a")
	if !ok || v2 != "1" {
		t.Fatalf("persistence: Get = (%q, %v)", v2, ok)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s, _ := New(tempPath(t))
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(2)
		k := string(rune('a' + i%26))
		go func() {
			defer wg.Done()
			_ = s.Set("b", k, "v")
		}()
		go func() {
			defer wg.Done()
			s.Get("b", k)
		}()
	}

	wg.Wait()

	if s.Len("b") == 0 {
		t.Fatal("expected non-empty store after concurrent writes")
	}
}

func TestSetRecordsTimestamp(t *testing.T) {
	s, _ := New(tempPath(t))
	before := time.Now().Unix()
	_ = s.Set("b", "k", "v")
	after := time.Now().Unix()

	s.mu.RLock()
	entry := s.data["b"]["k"]
	s.mu.RUnlock()

	if entry.SetAt < before || entry.SetAt > after {
		t.Fatalf("SetAt = %d, want between %d and %d", entry.SetAt, before, after)
	}
}
