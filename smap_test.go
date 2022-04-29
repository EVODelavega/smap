package smap_test

import (
	"testing"

	"github.com/EVODelavega/smap"
	"github.com/stretchr/testify/require"
)

func TestInitialisationWithMap(t *testing.T) {
	init := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	sm := smap.New(init)
	require.Equal(t, len(init), sm.Len())

	for k, v := range init {
		sv, ok := sm.Get(k)
		require.True(t, ok)
		require.Equal(t, v, sv)
	}
}

func TestInitNil(t *testing.T) {
	sm := smap.New[int, int64](nil)
	require.Equal(t, 0, sm.Len())

	k := 1
	sm.Set(k, 1)

	v, ok := sm.Get(k)
	require.True(t, ok)

	require.NotEqual(t, k, v)    // not the same type
	require.EqualValues(t, k, v) // same value
}

func TestAddRemove(t *testing.T) {
	init := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	sm := smap.New(init)

	keys := sm.Keys()
	require.Equal(t, len(init), len(keys)) // needs to return all keys

	for _, k := range keys {
		require.False(t, sm.CAS(k, init[k])) // shouldn't set anything
	}
	// OK, clone this map in its original state
	smClone := sm.Clone()
	require.Equal(t, len(init), smClone.Len())
	// now delete all keys
	sm.Delete(keys...)
	require.Equal(t, 0, sm.Len())

	// removing keys from a clone does not alter in any way the data of the clone
	require.Equal(t, len(init), smClone.Len())

	// Now set some random key on the original map
	sm.Set("four", 4) // new key
	sm.Set("one", 5)  // old key, new value
	// then merge the initial map
	sm.Merge(init, false)
	require.Equal(t, len(init)+1, sm.Len())

	newOne, ok := sm.Get("one")
	require.True(t, ok)
	require.NotEqual(t, newOne, init["one"])
	// now overwrite original keys
	sm.Merge(init, true)
	newOne, ok = sm.Get("one")
	require.True(t, ok)
	require.Equal(t, newOne, init["one"])
	// let's just ensure the map values are identical...
	raw := smClone.Raw()
	require.EqualValues(t, init, raw)
	// now ensure the "four" key can be set on the clone
	four, _ := sm.Get("four")
	require.True(t, smClone.CAS("four", four))
}

func TestIterNoSort(t *testing.T) {
	init := map[string]string{
		"foo": "bar",
		"bar": "foo",
		"zar": "car",
	}
	seen := map[string]struct{}{}
	sm := smap.New(init)
	it := sm.Iter(nil)
	for it.Next() {
		k, err := it.Key()
		require.NoError(t, err)
		v, err := it.Val()
		require.NoError(t, err)
		seen[k] = struct{}{}
		require.Equal(t, init[k], v)
	}
	it.Close()
	_, err := it.Key()
	require.Error(t, err)
	_, err = it.Val()
	require.Error(t, err)
}

func TestIterWithSort(t *testing.T) {
	init := map[int32]string{
		21: "twenty-one",
		1:  "one",
		3:  "three",
		2:  "two",
		15: "fifteen",
		11: "eleven",
	}
	expect := []int32{
		1, 2, 3, 11, 15, 21,
	}
	sm := smap.New(init)
	// sort ascending order
	it := sm.Iter(func(a, b int32) bool {
		return a < b
	})
	iterKeys := make([]int32, 0, len(init))
	for it.Next() {
		k, err := it.Key()
		require.NoError(t, err)
		v, err := it.Val()
		require.NoError(t, err)
		iterKeys = append(iterKeys, k)
		require.Equal(t, init[k], v)
	}
	it.Close()
	require.Equal(t, expect, iterKeys)
}
