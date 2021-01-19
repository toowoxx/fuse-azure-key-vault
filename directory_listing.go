package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html/atom"

	"bazil.org/fuse/fs"

	"github.com/andybalholm/cascadia"

	"golang.org/x/net/html"

	"github.com/pkg/errors"
)

type serverInfo struct {
	baseUrl   string
	lastInode uint64
}

type listingEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
	inode   uint64

	server    *serverInfo
	parent    *listingEntry
	children  []*listingEntry
	fileCount int

	fetchTime *time.Time

	fs.Node
}

type CustomNode html.Node

func (n CustomNode) AttrByName(name string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}

func (n CustomNode) RecursivelyFindNodeByTag(tag atom.Atom) *CustomNode {
	child := n.FirstChild
	for child != nil {
		if child.Type == html.ElementNode && child.DataAtom == tag {
			cn := CustomNode(*child)
			return &cn
		}
		node := CustomNode(*child).RecursivelyFindNodeByTag(tag)
		if node == nil {
			child = child.NextSibling
			continue
		} else {
			return node
		}
	}
	return nil
}

func (n CustomNode) RecursivelyFindNodeByTagReverse(tag atom.Atom) *CustomNode {
	parent := n.Parent
	for parent != nil {
		if parent.Type == html.ElementNode && parent.DataAtom == tag {
			cn := CustomNode(*parent)
			return &cn
		}
		parent = parent.Parent
	}
	return nil
}

func (l *listingEntry) parseDirectoryListing(root *html.Node) error {
	l.fileCount = 0

	entries := cascadia.MustCompile("tr > td > a").MatchAll(root)
	for _, entry := range entries {
		ancestorTrElement := CustomNode(*entry).RecursivelyFindNodeByTagReverse(atom.Tr)
		var imgElement *CustomNode
		if ancestorTrElement == nil {
			goto entryScan
		}
		imgElement = ancestorTrElement.RecursivelyFindNodeByTag(atom.Img)
		if imgElement != nil {
			altVal, exists := imgElement.AttrByName("alt")
			if exists && strings.EqualFold(altVal, "[parentdir]") {
				continue
			}
		}

	entryScan:
		entryName, exists := CustomNode(*entry).AttrByName("href")
		if !exists {
			continue
		}
		l.server.lastInode++
		isDir := strings.HasSuffix(entryName, "/")
		if isDir {
			entryName = strings.TrimRight(entryName, "/")
		}
		listingEntry := listingEntry{
			name:   entryName,
			inode:  l.server.lastInode,
			isDir:  isDir,
			server: l.server,
			parent: l,
		}
		l.children = append(l.children, &listingEntry)
		l.fileCount++
		if !listingEntry.isDir {
			listingEntry.fetchContentLength()
		}
	}
	return nil
}

func (l *listingEntry) GetPath() string {
	str := ""
	currentEntry := l

	for currentEntry != nil {
		if len(currentEntry.name) > 0 {
			str = currentEntry.name + "/" + str
		}
		currentEntry = currentEntry.parent
	}

	return strings.TrimRight(str, "/")
}

func (l *listingEntry) ToURLString() string {
	return fmt.Sprintf("%s/%s", l.server.baseUrl, l.GetPath())
}

func (l *listingEntry) retrieveDirectoryListing() error {
	p := l.ToURLString()
	if l.fetchTime != nil && l.fetchTime.Before(time.Now().Add(30*time.Second)) {
		return nil
	}

	URL, err := url.Parse(p)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not create URL for retrieving directory listing (path %s)", p))
	}

	resp, err := client().Get(URL.String())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not GET %s for directory listing", URL.String()))
	}

	HTML, err := html.Parse(resp.Body)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during read for GET %s", URL.String()))
	}

	err = l.parseDirectoryListing(HTML)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed parsing directory listing of %s", URL.String()))
	}

	now := time.Now()
	l.fetchTime = &now

	return nil
}

func (l *listingEntry) Find(name string) *listingEntry {
	for _, child := range l.children {
		if child.name == name {
			return child
		}
	}
	return nil
}

func (l *listingEntry) Download() ([]byte, error) {
	resp, err := client().Get(l.ToURLString())
	if err != nil {
		return nil, err
	}

	byt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return byt, nil
}

func (l *listingEntry) fetchContentLength() {
	resp, err := client().Head(l.ToURLString())
	if err != nil {
		return
	}

	contentLength := resp.Header.Get("Content-Length")
	if len(contentLength) == 0 {
		contentLength = resp.Header.Get("Size")
		if len(contentLength) == 0 {
			return
		}
	}

	size, _ := strconv.Atoi(contentLength)
	l.size = int64(size)
}

var (
	_ os.FileInfo = (*listingEntry)(nil)
)
