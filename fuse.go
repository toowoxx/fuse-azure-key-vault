package main

import (
	"context"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

func (l *listingEntry) fuseDirEntType() fuse.DirentType {
	switch {
	case l.isDir:
		return fuse.DT_Dir
	default:
		return fuse.DT_Dir
	}
}

func (l *listingEntry) toDirEnt() fuse.Dirent {
	return fuse.Dirent{
		Inode: l.inode,
		Type:  l.fuseDirEntType(),
		Name:  l.name,
	}
}

// FS implements the hello world file system.
type FS struct {
	RootEntry *Dir
}

func (fs FS) Root() (fs.Node, error) {
	return *fs.RootEntry, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	entry *listingEntry
}

func (d Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.entry.inode
	a.Mode = d.entry.Mode()
	a.Size = uint64(d.entry.Size())
	return nil
}

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	entry := d.entry.Find(name)
	if entry == nil {
		return nil, syscall.ENOENT
	}
	if entry.isDir {
		return Dir{entry}, nil
	} else {
		return File{entry}, nil
	}
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirs := []fuse.Dirent{
		{0, fuse.DT_Dir, "."},
		{0, fuse.DT_Dir, ".."},
	}

	err := d.entry.retrieveDirectoryListing()
	if err != nil {
		return nil, err
	}

	for _, child := range d.entry.children {
		dirs = append(dirs, child.toDirEnt())
	}

	return dirs, nil
}

// File implements both Node and Handle for the hello file.
type File struct {
	entry *listingEntry
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.entry.inode
	a.Mode = f.entry.Mode()
	a.Size = uint64(f.entry.Size())
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	return f.entry.Download()
}
