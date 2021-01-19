package main

import (
	"os"
	"time"
)

func (l *listingEntry) Name() string {
	return l.name
}

func (l *listingEntry) Size() int64 {
	if l.isDir {
		return int64(l.fileCount)
	}
	return l.size
}

func (l *listingEntry) Mode() os.FileMode {
	// read only by default
	bits := os.FileMode(0444)
	if l.isDir {
		bits |= os.ModeDir
		// Add execute bits for cd
		bits |= 0111
	}
	return bits
}

func (l *listingEntry) ModTime() time.Time {
	return l.modTime
}

func (l *listingEntry) IsDir() bool {
	return l.isDir
}

func (l *listingEntry) Sys() interface{} {
	return l
}
