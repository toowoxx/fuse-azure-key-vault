package main

import (
	"os"
	"time"
)

func (entry *listingEntry) Name() string {
	return entry.name
}

func (entry *listingEntry) Mode() os.FileMode {
	// read only by default
	bits := os.FileMode(0440)
	if entry.IsDir() {
		bits |= os.ModeDir
		// Add execute bits for cd
		bits |= 0110
	}
	return bits
}

func (entry *listingEntry) ModTime() time.Time {
	return entry.modTime
}

func (entry *listingEntry) Sys() interface{} {
	return entry
}
