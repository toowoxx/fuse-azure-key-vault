package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pkg/errors"

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

	keyVaultURLParam := flag.String("url", "", "URL of Azure Key Vault")
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

	if keyVaultURLParam == nil || len(*keyVaultURLParam) == 0 {
		usage()
		return
	}
	keyVaultURL := *keyVaultURLParam

	URL, err := url.Parse(keyVaultURL)
	if err != nil {
		fmt.Println(errors.Wrap(err, fmt.Sprintf("invalid URL \"%s\"", keyVaultURL)))
		os.Exit(int(syscall.EINVAL))
	}

	azKvClient := ConnectToKeyVault(URL.String())

	root := listingEntry{
		name:         "root",
		modTime:      time.Now(),
		inode:        1,
		parent:       nil,
		children:     nil,
		isRoot:       true,
		vaultClients: azKvClient,
		nextInode:    &atomic.Uint64{},
	}
	root.root = &root
	root.nextInode.Add(root.inode)

	handleStopsAndCrashes()
	defer func() {
		if r := recover(); r != nil {
			_ = conn.Close()
			_ = fuse.Unmount(mountDir)
			panic(r)
		}
	}()

	log.Println("Mounting", keyVaultURL, "on", mountDir)
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
