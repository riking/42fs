package fscore

import (
	"fmt"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
	"sync"
	"os"
	"path/filepath"
	"strings"
)

type LocalDir struct {
	fs42      *FS42
	Root      string
	LocalNode

	lock      sync.Mutex
	pathCache map[string]*LocalNode
	openFiles map[int]*LocalFile
}

func NewLocalDir(fs42 *FS42, root string) *LocalDir {
	md := &LocalDir{
		fs42:      fs42,
		Root:      root,
		pathCache: make(map[string]*LocalNode),
		openFiles: make(map[int]*LocalFile),
	}
	md.LocalNode = LocalNode{md: md, Path: "."}
	md.pathCache[""] = &md.LocalNode
	md.pathCache["."] = &md.LocalNode

	var _ fs.NodeAccesser = &md.LocalNode
	var _ fs.NodeForgetter = &md.LocalNode
	var _ fs.NodeGetattrer = &md.LocalNode
	var _ fs.NodeGetxattrer = &md.LocalNode
	var _ fs.NodeListxattrer = &md.LocalNode
	var _ fs.NodeOpener = &md.LocalNode
	var _ fs.NodeReadlinker = &md.LocalNode
	var _ fs.NodeStringLookuper = &md.LocalNode
	return md
}

func (md *LocalDir) nodeFor(parent *LocalNode, name string) *LocalNode {
	relName := parent.JoinRelative(name)
	md.lock.Lock()
	defer md.lock.Unlock()

	existing, ok := md.pathCache[relName]
	if ok {
		return existing
	}
	ln := &LocalNode{md: md, Path: relName}
	md.pathCache[relName] = ln
	return ln
}

func (md *LocalDir) forgetNode(name string) {
	md.lock.Lock()
	defer md.lock.Unlock()

	_, ok := md.pathCache[name]
	if ok {
		delete(md.pathCache, name)
	}
}

type LocalNode struct {
	md   *LocalDir
	Path string
}

func (d *LocalNode) FullPath() string {
	return fmt.Sprintf("%s/%s", d.md.Root, d.Path)
}

func (d *LocalNode) Join(file string) string {
	file = filepath.Clean(file)
	if strings.HasPrefix(file, "..") {
		return "/BAD_PATH"
	}
	return fmt.Sprintf("%s/%s/%s", d.md.Root, d.Path, file)
}

func (d *LocalNode) JoinRelative(file string) string {
	file = filepath.Clean(file)
	if strings.HasPrefix(file, "..") {
		return "/BAD_PATH"
	}
	return fmt.Sprintf("%s/%s", d.Path, file)
}

func (d *LocalNode) Forget() {
	d.md.lock.Lock()
	defer d.md.lock.Unlock()

	if d.md.pathCache[d.Path] == d {
		delete(d.md.pathCache, d.Path)
	}
}

// methods

func (d *LocalNode) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	var a fuse.Attr
	err := d.Attr(ctx, &a)
	if err != nil {
		return err
	}

	resp.Attr = a
	return nil
}

type LocalFile struct {
	ln     *LocalNode
	fd     int
	dirty  bool

	osFile *os.File
}
