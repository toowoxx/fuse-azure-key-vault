package main

import (
	"os"
	"time"

	"bazil.org/fuse/fs"
)

type listingEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
	inode   uint64

	parent    *listingEntry
	children  []*listingEntry
	fileCount int

	fetchTime *time.Time

	fs.Node
}

var (
	_ os.FileInfo = (*listingEntry)(nil)
)
