package fscore

import (
	"fmt"

	fgrpc "github.com/riking/42fs/grpc"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
	"sync"
)

type UserDir struct {
	fs42      *FS42
	curCon    fgrpc.UserConnection
	login     string
	RemoteNode

	lock      sync.Mutex
	pathCache map[string]*RemoteNode
	openFiles map[uint64]*RemoteFile
}

func NewUserDir(fs42 *FS42, login *fgrpc.LoginInfo) *UserDir {
	d := &UserDir{
		fs42:   fs42,
		curCon: nil, // TODO
		login:  login.Login,
	}
	d.pathCache = make(map[string]*RemoteNode)
	d.openFiles = make(map[uint64]*RemoteFile)

	d.RemoteNode = RemoteNode{ud: d, Path: ""}
	d.pathCache[""] = &d.RemoteNode

	var _ fs.NodeAccesser = &d.RemoteNode
	var _ fs.NodeForgetter = &d.RemoteNode
	var _ fs.NodeGetattrer = &d.RemoteNode
	var _ fs.NodeGetxattrer = &d.RemoteNode
	var _ fs.NodeListxattrer = &d.RemoteNode
	var _ fs.NodeOpener = &d.RemoteNode
	var _ fs.NodeReadlinker = &d.RemoteNode
	var _ fs.NodeStringLookuper = &d.RemoteNode
	return d
}

func (ud *UserDir) nodeFor(parent *RemoteNode, name string) *RemoteNode {
	fullName := parent.Join(name)
	ud.lock.Lock()
	defer ud.lock.Unlock()

	existing, ok := ud.pathCache[fullName]
	if ok {
		return existing
	}
	rn := &RemoteNode{ud: ud, Path: fullName}
	ud.pathCache[fullName] = rn
	return rn
}

func (ud *UserDir) conn() fgrpc.UserConnection {
	return ud.curCon
}

type RemoteNode struct {
	ud   *UserDir
	Path string
}

func (d *RemoteNode) Join(name string) string {
	return fmt.Sprintf("%s/%s", d.Path, name)
}

func (d *RemoteNode) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return d.ud.conn().Access(ctx, d.Path, req.Mask)
}

func (d *RemoteNode) Attr(ctx context.Context, a *fuse.Attr) error {
	st, err := d.ud.conn().Stat(ctx, d.Path)
	if err != nil {
		return err
	}

	a.Inode = st.INode
	a.Size = st.Size
	a.Blocks = st.Blocks
	a.Atime = st.Mtime
	a.Mtime = st.Mtime
	a.Ctime = st.Ctime
	a.Crtime = st.BirthTime
	a.Nlink = st.Nlink
	a.Uid = st.Uid
	a.Gid = st.Gid
	a.BlockSize = st.BlockSize
	a.Mode = st.Mode

	return nil
}

func (d *RemoteNode) Forget() {
	d.ud.lock.Lock()
	defer d.ud.lock.Unlock()

	if d.ud.pathCache[d.Path] == d {
		delete(d.ud.pathCache, d.Path)
	}
}

func (d *RemoteNode) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	var a fuse.Attr
	err := d.Attr(ctx, &a)
	if err != nil {
		return err
	}

	resp.Attr = a
	return nil
}

func (d *RemoteNode) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	b, err := d.ud.conn().Getxattr(ctx, d.Path, req.Name, req.Size, req.Position)
	if err != nil {
		return err
	}
	resp.Xattr = b
	return nil
}

func (d *RemoteNode) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	b, err := d.ud.conn().Listxattr(ctx, d.Path, req.Size, req.Position)
	if err != nil {
		return err
	}
	resp.Xattr = b
	return nil
}

func (d *RemoteNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	err := d.ud.conn().LookupExists(ctx, d.Join(name))
	if err != nil {
		return nil, err
	}
	return d.ud.nodeFor(d, name), nil
}

func (d *RemoteNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	oflags := fgrpc.ToAgnostic(req.Flags)
	rflags, fd, err := d.ud.conn().Open(ctx, d.Path, req.Dir, oflags)
	if err != nil {
		return nil, err
	}
	resp.Flags = rflags
	rFile := &RemoteFile{rn: d, fd: fd}

	d.ud.lock.Lock()
	defer d.ud.lock.Unlock()
	d.ud.openFiles[fd] = rFile
	return rFile, nil
}

func (d *RemoteNode) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	return d.ud.conn().Readlink(ctx, d.Path)
}

type RemoteFile struct {
	rn *RemoteNode
	fd uint64
}

func (f *RemoteFile) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	cEnts, err := f.rn.ud.conn().ReadDir(ctx, f.fd)
	if err != nil {
		return nil, err
	}
	fEnts := make([]fuse.Dirent, len(cEnts))
	for i := range cEnts {
		fEnts[i].Inode = cEnts[i].Inode
		fEnts[i].Name = cEnts[i].Name
		fEnts[i].Type = fuse.DirentType(cEnts[i].Type)
	}
	return fEnts, nil
}

func (f *RemoteFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	var cReq fgrpc.ReadRequest
	cReq.Dir = req.Dir
	cReq.FD = f.fd
	cReq.FileFlags = req.FileFlags
	cReq.Offset = req.Offset
	cReq.Size = req.Size
	b, err := f.rn.ud.conn().ReadFrom(ctx, &cReq)
	if err != nil {
		return err
	}
	resp.Data = b
	return nil
}

func (f *RemoteFile) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	f.rn.ud.lock.Lock()
	delete(f.rn.ud.openFiles, f.fd)
	f.rn.ud.lock.Unlock()

	return f.rn.ud.conn().Close(ctx, f.fd)
}
