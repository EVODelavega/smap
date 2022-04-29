package smap

import (
	"errors"
	"sort"
	"sync"
)

type sMap[K comparable, V any] struct {
	mu *sync.RWMutex
	m  map[K]V
}

type sMapIter[K comparable, V any] struct {
	l    sync.Locker
	i    int
	keys []K
	k    K
	v    V
	m    *sMap[K, V]
}

// New creates a new sync-safe map
func New[K comparable, V any](init map[K]V) *sMap[K, V] {
	r := &sMap[K, V]{
		mu: &sync.RWMutex{},
		m:  map[K]V{},
	}
	// initialise
	r.Merge(init, true) // overwrite doesn't make a difference but we can skip pointless lookups
	return r
}

// Len returns underlying map length
func (s *sMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Merge merges a given map into this type
func (s *sMap[K, V]) Merge(m map[K]V, overwrite bool) {
	if len(m) == 0 {
		return
	}
	s.mu.Lock()
	for k, v := range m {
		if !overwrite {
			if _, ok := s.m[k]; ok {
				continue
			}
		}
		s.m[k] = v
	}
	s.mu.Unlock()
}

// Clone creates a copy
func (s *sMap[K, V]) Clone() *sMap[K, V] {
	s.mu.RLock()
	r := New[K, V](s.m) // create new instance
	s.mu.RUnlock()
	return r
}

// Get simply gets the value for a given key, returns false if the key doesn't exist
func (s *sMap[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	v, ok := s.m[k]
	s.mu.RUnlock()
	return v, ok
}

// Set sets a value for a given key (overwrites existing value)
func (s *sMap[K, V]) Set(k K, v V) {
	s.mu.Lock()
	s.m[k] = v
	s.mu.Unlock()
}

// CAS is a simple Check And Set, returns false if the key was not set
func (s *sMap[K, V]) CAS(k K, v V) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[k]; ok {
		return false
	}
	s.m[k] = v
	return true
}

// Raw returns a copy of the underlying map as a standard map[K]V
func (s *sMap[K, V]) Raw() map[K]V {
	s.mu.RLock()
	ret := make(map[K]V, len(s.m))
	for k, v := range s.m {
		ret[k] = v
	}
	s.mu.RUnlock()
	return ret
}

// Delete deletes one or more of the keys. Non-existing keys are a no-op as with a normal map
func (s *sMap[K, V]) Delete(keys ...K) {
	s.mu.Lock()
	for _, k := range keys {
		delete(s.m, k)
	}
	s.mu.Unlock()
}

// Keys returns a slice of all keys
func (s *sMap[K, V]) Keys() []K {
	s.mu.RLock()
	ks := make([]K, 0, len(s.m))
	for k := range s.m {
		ks = append(ks, k)
	}
	s.mu.RUnlock()
	return ks
}

// Iter returns an iterator, iteration is non-deterministic like a normal map, unless
// the optional sort function is provided, in which case the keys will be sorted using sort.SliceStable
// After iterating over the values, Close must be called!
func (s *sMap[K, V]) Iter(f func(a, b K) bool) *sMapIter[K, V] {
	iter := &sMapIter[K, V]{
		l: s.mu.RLocker(),
		m: s,
		i: 0,
	}
	iter.l.Lock() // acquire lock already
	keys := s.Keys()
	if f != nil {
		sort.SliceStable(keys, func(i, j int) bool {
			return f(keys[i], keys[j])
		})
	}
	iter.keys = keys
	return iter
}

// Next moves the iterator to the next element in the map, returns false if we already reached the end
func (i *sMapIter[K, V]) Next() bool {
	if i.i >= len(i.keys) {
		return false
	}
	// set key/value
	i.k = i.keys[i.i]
	i.v = i.m.m[i.k]
	i.i++ // move index
	return true
}

// Key returns current key
func (i *sMapIter[K, V]) Key() (K, error) {
	var k K
	if i.keys == nil {
		return k, errors.New("iterator closed")
	}
	return i.k, nil
}

// Val returns current value
func (i *sMapIter[K, V]) Val() (V, error) {
	var v V
	if i.keys == nil {
		return v, errors.New("iterator closed")
	}
	return i.v, nil
}

// Close releases the iterator
func (i *sMapIter[K, V]) Close() {
	var (
		k K
		v V
	)
	// clear all fields
	i.keys = nil
	i.k = k
	i.v = v
	i.i = 0
	i.m = nil
	// release lock
	i.l.Unlock()
}
