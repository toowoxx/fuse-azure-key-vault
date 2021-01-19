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

type fsRootNode struct{}

func (f fsRootNode) Name() string {
	return ""
}

func (f fsRootNode) Size() int64 {
	return 0
}

func (f fsRootNode) Mode() os.FileMode {
	return os.ModeDir
}

func (f fsRootNode) ModTime() time.Time {
	return time.Now()
}

func (f fsRootNode) IsDir() bool {
	return true
}

func (f fsRootNode) Sys() interface{} {
	return f
}

var (
	_ os.FileInfo = (*fsRootNode)(nil)
)
