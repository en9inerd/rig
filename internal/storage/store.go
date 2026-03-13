package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry holds a value and the time it was written.
type Entry struct {
	Value string `json:"value"`
	SetAt int64  `json:"set_at"` // unix seconds
}

// Store is a lightweight key-value store backed by a single JSON file.
// Writes are atomic (temp file + rename) so a crash can never leave a
// partially written file. The entire dataset lives in memory; the file
// is rewritten on every mutation.
type Store struct {
	path string
	mu   sync.RWMutex
	data map[string]map[string]Entry // bucket → key → entry
}

// New opens (or creates) a store at path. The parent directory is
// created automatically if it does not exist.
func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	s := &Store{
		path: path,
		data: make(map[string]map[string]Entry),
	}

	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &s.data); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) bucket(name string) map[string]Entry {
	b, ok := s.data[name]
	if !ok {
		b = make(map[string]Entry)
		s.data[name] = b
	}
	return b
}

// Get returns the value for key in bucket. The second return value
// reports whether the key was found.
func (s *Store) Get(bucket, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.data[bucket]
	if !ok {
		return "", false
	}
	e, ok := b[key]
	return e.Value, ok
}

// Has reports whether key exists in bucket.
func (s *Store) Has(bucket, key string) bool {
	_, ok := s.Get(bucket, key)
	return ok
}

// Set writes a key-value pair into bucket and flushes to disk.
func (s *Store) Set(bucket, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bucket(bucket)[key] = Entry{
		Value: value,
		SetAt: time.Now().Unix(),
	}
	return s.flush()
}

// SetBatch writes multiple key-value pairs into bucket in a single
// flush. More efficient than calling Set in a loop.
func (s *Store) SetBatch(bucket string, pairs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b := s.bucket(bucket)
	now := time.Now().Unix()
	for k, v := range pairs {
		b[k] = Entry{Value: v, SetAt: now}
	}
	return s.flush()
}

// Delete removes a key from bucket and flushes to disk.
func (s *Store) Delete(bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.data[bucket]
	if !ok {
		return nil
	}
	if _, exists := b[key]; !exists {
		return nil
	}
	delete(b, key)
	return s.flush()
}

// ForEach calls fn for every key-value pair in bucket. Iteration stops
// early if fn returns a non-nil error (which is propagated).
func (s *Store) ForEach(bucket string, fn func(key, value string) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k, e := range s.data[bucket] {
		if err := fn(k, e.Value); err != nil {
			return err
		}
	}
	return nil
}

// Keys returns all keys in bucket.
func (s *Store) Keys(bucket string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b := s.data[bucket]
	keys := make([]string, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of keys in bucket.
func (s *Store) Len(bucket string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data[bucket])
}

// Prune deletes all entries in bucket whose SetAt is older than cutoff
// and flushes to disk. Returns the number of entries removed.
func (s *Store) Prune(bucket string, cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.data[bucket]
	if !ok {
		return 0, nil
	}

	threshold := cutoff.Unix()
	var removed int
	for k, e := range b {
		if e.SetAt < threshold {
			delete(b, k)
			removed++
		}
	}

	if removed > 0 {
		return removed, s.flush()
	}
	return 0, nil
}

// flush writes the store data to disk atomically.
// Caller must hold s.mu.
func (s *Store) flush() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}
