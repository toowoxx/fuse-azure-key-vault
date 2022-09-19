package main

import (
	"context"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

func (entry *listingEntry) fuseDirEntType() fuse.DirentType {
	switch {
	case entry.IsDir():
		return fuse.DT_Dir
	default:
		return fuse.DT_Dir
	}
}

func (entry *listingEntry) toDirEnt() fuse.Dirent {
	return fuse.Dirent{
		Inode: entry.inode,
		Type:  entry.fuseDirEntType(),
		Name:  entry.name,
	}
}

type FS struct {
	RootEntry *Dir
}

func (fs FS) Root() (fs.Node, error) {
	return *fs.RootEntry, nil
}

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
	entry := d.entry.Find(name, ctx)
	if entry == nil {
		return nil, syscall.ENOENT
	}
	if entry.IsDir() {
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

	err := d.entry.retrieveDirectoryListing(ctx)
	if err != nil {
		return nil, err
	}

	for _, child := range d.entry.children {
		dirs = append(dirs, child.toDirEnt())
	}

	return dirs, nil
}

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
	return f.entry.Download(ctx)
}
