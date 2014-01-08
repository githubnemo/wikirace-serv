package main

import (
	"encoding/json"
	"github.com/peterbourgon/diskv"
)

// Persistent storage with marhsalling.
//
// All crucial actions on this store (Write, Erase, ...) are locked
// and safe for concurrent access.
type Store struct {
	*diskv.Diskv
}

func NewStore(dir string) *Store {
	s := diskv.New(diskv.Options{
		BasePath: dir,
	})

	return &Store{s}
}

func (g *Store) PutMarshal(hash string, v interface{}) error {
	bytes, err := json.MarshalIndent(v, "", "  ")

	if err != nil {
		return err
	}

	// XXX: circumvent a bug in diskv when the new content has fewer bytes
	// than the new
	g.Diskv.Erase(hash)

	return g.Diskv.Write(hash, bytes)
}

func (g *Store) GetMarshal(hash string, v interface{}) error {
	bytes, err := g.Diskv.Read(hash)

	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, v)
}

func (g *Store) Contains(hash string) bool {
	for key := range g.Keys() {
		if key == hash {
			return true
		}
	}
	return false
}
