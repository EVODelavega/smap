# smap

A generic map type that is safe for concurrent use. Kind of like the old `sync.Map` type, but without the type assertions.

## Usage

Crating a new instance is pretty straightforward:

```go
// from an existing map variable:
safe := smap.New(oldMap)

// an empty map with certain types:
blank := smap.New[error, uint64](nil) // underlying map[error]uint64
```

You can perform all standard operations on the map, through the interface:

```go
safe.Set(key, value) // set a new key
val, ok := safe.Get(key) // get a value, ok == false if key doesn't exist
safe.Delete(key) // delete one or more keys, function is variadic
```

Along with some creature-comfort methods:

```go
// clone the map
cpy := safe.Clone()
// Get the underlying data as a normal map[K]V
unsafe := safe.Raw()
// CAS (Check And Set)
wasSet := safe.CAS(k, v) // returns false if the key already exists

// merge maps with or without overwriting existing keys
safe.Merge(anotherMap, false) // don't overwrite keys that already exist
safe.Merge(anotherMap, true) // overwrite existing keys

// Get the keys from the map
keys := safe.Keys() // return all keys from the map
```

## Iterator

As a nice-to have, this map type has an iterator. Basic usage would be:

```go
it := safe.Iter(nil)
for it.Next() {
    key, err := it.Key()
    val, err := it.Val()
    // check errors, use key/value as needed
}
it.Close() // must be called to release the internal RLock
```

The iterator, like a regular loop over a map is non-deterministic. To ensure the keys are iterated over in the same/right order, you can pass in a sort function:

```go
safe := smap.New[uint64, string](nil)
// populate map
it := safe.Iter(func(a, b uint64) bool {
    return a < b // ascending order
})

for it.Next() {
}
it.Close()
```

This callback is used inside the `slice.SortStable` callback, and so it ensures map iteration is consistent no matter what.
While an iterator is open, no writes to the map can happen (otherwise it wouldn't be safe for concurrent use after all).
