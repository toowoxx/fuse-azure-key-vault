package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var conn *fuse.Conn
var mountDir string
var isExiting = false

func handleStopsAndCrashes() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGSEGV)
	go func() {
		s := <-sigChan
		log.Println("Signal received:", s)
		if isExiting {
			log.Println("Force-quitting")
			os.Exit(1)
		}
		isExiting = true
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			log.Println("Closing connection...")
			err := conn.Close()
			if err != nil {
				log.Println("Error while closing connection")
				log.Fatal(err)
			}
			log.Println("Connection closed.")
			wg.Done()
		}()
		log.Println("Unmounting", mountDir)
		err := fuse.Unmount(mountDir)
		wg.Wait()
		if err != nil {
			log.Println("Error while exiting")
			log.Fatal(err)
		} else {
			log.Println("Exiting.")
			log.Println()
			os.Exit(0)
		}
	}()
}

func main() {
	var err error

	// serverBaseUrlP := flag.String("url", "", "Base URL to mount")
	flag.Parse()
	mountDir = flag.Arg(0)

	if len(mountDir) == 0 {
		usage()
		return
	}

	if !dirExists(mountDir) {
		fmt.Println(fmt.Errorf("%s: not found or not a directory", mountDir))
		os.Exit(int(syscall.ENOENT))
	}

	root := listingEntry{
		name:      "",
		isDir:     true,
		size:      0,
		modTime:   time.Now(),
		inode:     1,
		parent:    nil,
		children:  nil,
		fileCount: 0,
	}

	handleStopsAndCrashes()
	defer func() {
		if r := recover(); r != nil {
			_ = conn.Close()
			_ = fuse.Unmount(mountDir)
			panic(r)
		}
	}()

	conn, err = fuse.Mount(
		mountDir,
		fuse.FSName("azure-key-vault"),
		fuse.Subtype("azkv"),
		fuse.AllowNonEmptyMount(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	err = fs.Serve(conn, FS{
		RootEntry: &Dir{
			entry: &root,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = root
	_ = err
}

func dirExists(path string) bool {
	stat, err := os.Stat(path)
	if err == nil {
		return stat.IsDir()
	}
	return false
}

func usage() {
	fmt.Printf("Usage: %s [options...] <mount point>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}
