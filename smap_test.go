package smap_test

import (
	"fmt"
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

func TestFilter(t *testing.T) {
	t.Run("Filter by key", testFilterKeys)
	t.Run("Filter by values", testFilterValues)
	t.Run("Filter by key and value", testFilterKV)
}

func testFilterKeys(t *testing.T) {
	init := map[int32]string{
		21: "twenty-one",
		1:  "one",
		3:  "three",
		2:  "two",
		15: "fifteen",
		11: "eleven",
	}
	expect := []int32{
		2,
	}
	sm := smap.New(init)
	// sort ascending order
	filter := sm.Filter(func(k int32, _ string) bool {
		return (k%2 == 0)
	})
	// should only yield a single value
	require.Equal(t, len(expect), len(filter))
	require.Equal(t, filter[expect[0]], init[expect[0]])

	// now the same must hold true for odd values
	expect = []int32{
		1, 3, 11, 15, 21,
	}
	filter = sm.Filter(func(k int32, _ string) bool {
		return (k%2 != 0)
	})
	// should only yield a single value
	require.Equal(t, len(expect), len(filter))
	for i := range expect {
		require.Equal(t, filter[expect[i]], init[expect[i]])
	}
}

func testFilterValues(t *testing.T) {
	init := map[string]int32{
		"twenty-one": 21,
		"one":        1,
		"three":      3,
		"two":        2,
		"fifteen":    15,
		"eleven":     11,
	}
	expect := []int32{
		2,
	}
	sm := smap.New(init)
	// sort ascending order
	filter := sm.Filter(func(_ string, v int32) bool {
		return (v%2 == 0)
	})
	// should only yield a single value
	require.Equal(t, len(expect), len(filter))

	// now the same must hold true for odd values
	expect = []int32{
		1, 3, 11, 15, 21,
	}
	filter = sm.Filter(func(_ string, v int32) bool {
		return (v%2 != 0)
	})
	// should only yield a single value
	require.Equal(t, len(expect), len(filter))
}

func testFilterKV(t *testing.T) {
	init := map[int]uint{
		1: 1,
		3: 2,
		5: 10,
		8: 64,
		9: 24,
	}
	sm := smap.New(init)
	// we're filtering values where v%k == 0
	expect := map[int]uint{
		1: 1,
		5: 10,
		8: 64,
	}

	filter := sm.Filter(func(k int, v uint) bool {
		uk := uint(k)
		return (v%uk == 0)
	})
	require.Equal(t, len(expect), len(filter))
	require.EqualValues(t, expect, filter)
}

type iter interface {
	Next() bool
	Prev() bool
	End() bool
	Rewind() bool
	Close()
}

func TestRIter(t *testing.T) {
	init := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
		"e": 5,
	}
	sm := smap.New(init)
	it := sm.RIter(func(a, b string) bool {
		return a < b
	})
	rev := sm.RIter(func(a, b string) bool {
		return a > b
	})
	asc := make([]string, 0, len(init))
	for it.Next() {
		k, _ := it.Key()
		v, _ := it.Val()
		fmt.Printf("%s => %v\n", k, v)
		asc = append(asc, k)
	}
	// just to make sure we can edit while iterating
	sm.Delete(asc[0])
	// make sure end of reverse == first of ascending
	require.True(t, rev.End())
	k, _ := rev.Key()
	require.True(t, rev.Rewind())
	_, ok := sm.Get(k)
	// value was removed
	require.False(t, ok)
	require.True(t, it.End())
	for rev.Next() {
		k, _ := rev.Key()
		v, _ := rev.Val()
		ak, _ := it.Key()
		av, _ := it.Val()
		_ = it.Prev()
		require.Equal(t, k, ak)
		require.Equal(t, v, av)
	}
	iters := []iter{
		it, rev,
	}
	for _, i := range iters {
		// closing is mostly for show, but marks an iterator as no longer in use
		i.Close()
		// none of these calls should work
		require.False(t, i.Next())
		require.False(t, i.Prev())
		require.False(t, i.End())
		require.False(t, i.Rewind())
	}
	_, err := it.Key()
	require.Error(t, err)
	_, err = rev.Val()
	require.Error(t, err)
}
