package fscore

import (
	"os"
	"fmt"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/sys/unix"
	"golang.org/x/net/context"
)

// fileMode returns a Go os.FileMode from a Unix mode.
func fileMode(unixMode uint32) os.FileMode {
	mode := os.FileMode(unixMode & 0777)
	switch unixMode & unix.S_IFMT {
	case unix.S_IFREG:
	// nothing
	case unix.S_IFDIR:
		mode |= os.ModeDir
	case unix.S_IFCHR:
		mode |= os.ModeCharDevice | os.ModeDevice
	case unix.S_IFBLK:
		mode |= os.ModeDevice
	case unix.S_IFIFO:
		mode |= os.ModeNamedPipe
	case unix.S_IFLNK:
		mode |= os.ModeSymlink
	case unix.S_IFSOCK:
		mode |= os.ModeSocket
	default:
		// no idea
		mode |= os.ModeDevice
	}
	if unixMode&unix.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if unixMode&unix.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	if unixMode&unix.S_ISVTX != 0 {
		mode |= os.ModeSticky
	}
	// Not handled:
	// os.ModeAppend
	// os.ModeExclusive
	// os.ModeTemporary
	return mode
}

// unixMode returns a local unix mode from an os.FileMode
func unixMode(osMode os.FileMode) uint32 {
	mode := uint32(osMode & 0777)
	switch osMode & os.ModeType {
	case 0:
		mode |= unix.S_IFREG
	case os.ModeDir:
		mode |= unix.S_IFDIR
	case os.ModeDevice | os.ModeCharDevice:
		mode |= unix.S_IFCHR
	case os.ModeDevice:
		mode |= unix.S_IFBLK
	case os.ModeNamedPipe:
		mode |= unix.S_IFIFO
	case os.ModeSymlink:
		mode |= unix.S_IFLNK
	case os.ModeSocket:
		mode |= unix.S_IFSOCK
	default:
		// no idea
		mode |= unix.S_IFBLK
	}
	if osMode & os.ModeSetuid != 0 {
		mode |= unix.S_ISUID
	}
	if osMode & os.ModeSetgid != 0 {
		mode |= unix.S_ISGID
	}
	if osMode & os.ModeSticky != 0 {
		mode |= unix.S_ISVTX
	}
	return mode
}

// unixCreateMode only returns the mode bits that have an effect on file creation
func unixCreateMode(osMode os.FileMode) uint32 {
	mode := uint32(osMode & 0777)
	if osMode & os.ModeSetuid != 0 {
		mode |= unix.S_ISUID
	}
	if osMode & os.ModeSetgid != 0 {
		mode |= unix.S_ISGID
	}
	if osMode & os.ModeSticky != 0 {
		mode |= unix.S_ISVTX
	}
	return mode
}

func toTimeval(t time.Time) unix.Timeval {
	return unix.NsecToTimeval(t.UnixNano())
}

func (d *LocalNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	err := unix.Access(d.Join(name), unix.F_OK)
	if err != nil {
		return nil, err
	}
	return d.md.nodeFor(d, name), nil
}

func (d *LocalNode) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return unix.Access(d.FullPath(), req.Mask)
}

func (d *LocalNode) Link(ctx context.Context, req *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
	oldLN, ok := old.(*LocalNode)
	if !ok {
		return nil, fuse.Errno(unix.EBADF)
	}
	err := unix.Link(oldLN.FullPath(), d.Join(req.NewName))
	if err != nil {
		return nil, err
	}
	newLn := d.md.nodeFor(d, req.NewName)
	return newLn, nil
}

func (d *LocalNode) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fs.Node, error) {
	err := unix.Symlink(req.Target, d.Join(req.NewName))
	if err != nil {
		return nil, err
	}
	newLn := d.md.nodeFor(d, req.NewName)
	return newLn, nil
}

func (d *LocalNode) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	fullPath := d.FullPath()
	b := make([]byte, 0, 64)
	var l int
	var err error
	for {
		l, err = unix.Readlink(fullPath, b)
		if err != nil {
			return "", err
		}
		if l <= cap(b) {
			break
		}
		b = make([]byte, 0, cap(b) * 2)
	}
	return string(b[:l]), nil
}

func (d *LocalNode) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Dir {
		return unix.Rmdir(d.Join(req.Name))
	} else {
		return unix.Unlink(d.Join(req.Name))
	}
}

func (d *LocalNode) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	newLN, ok := newDir.(*LocalNode)
	if !ok {
		return fuse.Errno(unix.EBADF)
	}
	return unix.Rename(d.Join(req.OldName), newLN.Join(req.NewName))
}

func (d *LocalNode) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	var a fuse.Attr
	err := d.Attr(ctx, &a)
	if err != nil {
		return err
	}
	fullPath := d.FullPath()

	var finalErr error
	addErr := func(e error) {
		if finalErr == nil {
			finalErr = e
		}
	}

	if req.Valid.Size() {
		err = unix.Truncate(fullPath, int64(req.Size))
		addErr(err)
	}
	if req.Valid.Mode() {
		err = os.Chmod(fullPath, req.Mode)
		if err != nil {
			addErr(err.(*os.PathError).Err)
		}
	}
	if req.Valid.Uid() || req.Valid.Gid() {
		// haha, nice joke. no. not allowed.
		addErr(fuse.EPERM)
	}
	addErr(d.setattrPlatform(ctx, req, resp))
	if req.Valid.Mtime() || req.Valid.Atime() || req.Valid.MtimeNow() || req.Valid.AtimeNow() {
		var times [2]unix.Timeval
		times[0] = toTimeval(a.Atime)
		times[1] = toTimeval(a.Mtime)
		if req.Valid.Atime() {
			if req.Valid.AtimeNow() {
				times[0] = toTimeval(time.Now())
			} else {
				times[0] = toTimeval(req.Atime)
			}
		}
		if req.Valid.Mtime() {
			if req.Valid.MtimeNow() {
				times[0] = toTimeval(time.Now())
			} else {
				times[0] = toTimeval(req.Mtime)
			}
		}
		// must open the file to get NOFOLLOW semantics
		fd, err := unix.Open(fullPath, unix.O_NOFOLLOW | unix.O_RDONLY, 0)
		if err == nil {
			err = unix.Futimes(fd, times[:])
			unix.Close(fd)
		}
		addErr(err)
	}

	return finalErr
}


func (d *LocalNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (h fs.Handle, err error) {
	defer func() {
		fmt.Println("open", d.Path, req.Flags, h, err)
	}()

	req.Flags &^= unix.O_NONBLOCK
	fd, err := unix.Open(d.FullPath(), int(req.Flags), 0)
	if err != nil {
		return nil, err
	}
	lFile := &LocalFile{ln: d, fd: fd}

	d.md.lock.Lock()
	defer d.md.lock.Unlock()
	d.md.openFiles[fd] = lFile
	return lFile, nil
}

func (d *LocalNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	var oldUmask int
	if req.Umask != 0 {
		oldUmask = unix.Umask(int(unixCreateMode(req.Umask)))
	}
	req.Flags &^= unix.O_NONBLOCK
	fd, err := unix.Open(d.Join(req.Name), int(req.Flags), unixCreateMode(req.Mode))
	if err != nil {
		return nil, nil, err.(*os.PathError).Err
	}
	if req.Umask != 0 {
		unix.Umask(oldUmask)
	}
	newLn := d.md.nodeFor(d, req.Name)
	newLf := &LocalFile{
		fd: fd,
		ln: newLn,
	}

	d.md.lock.Lock()
	defer d.md.lock.Unlock()
	d.md.openFiles[fd] = newLf
	return newLn, newLf, nil
}

func (d *LocalNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	var oldUmask int
	if req.Umask != 0 {
		oldUmask = unix.Umask(int(unixCreateMode(req.Umask)))
	}
	err := unix.Mkdir(d.Join(req.Name), unixCreateMode(req.Mode))
	if req.Umask != 0 {
		unix.Umask(oldUmask)
	}
	if err != nil {
		return nil, err
	}
	newLn := d.md.nodeFor(d, req.Name)
	return newLn, nil
}

// type LocalFile

func (f *LocalFile) getOSFile() (*os.File, error) {
	if f.osFile != nil {
		return f.osFile, nil
	}
	newFD, err := unix.Dup(f.fd)
	if err != nil {
		return nil, err
	}
	f.osFile = os.NewFile(uintptr(newFD), f.ln.Path)
	return f.osFile, nil
}

func (f *LocalFile) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	// no-op?
	return nil
}

func (f *LocalFile) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("readdir", f.ln.Path)
	of, err := f.getOSFile()
	if err != nil {
		return nil, err
	}
	names, err := of.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	var stat_t unix.Stat_t
	fuseEnts := make([]fuse.Dirent, len(names))
	for i, v := range names {
		fuseEnts[i].Name = v
		err = unix.Lstat(f.ln.Join(v), &stat_t)
		if err != nil {
			continue
		}
		switch {
		case stat_t.Mode&unix.S_IFSOCK != 0:
			fuseEnts[i].Type = fuse.DT_Socket
		case stat_t.Mode&unix.S_IFLNK != 0:
			fuseEnts[i].Type = fuse.DT_Link
		case stat_t.Mode&unix.S_IFREG != 0:
			fuseEnts[i].Type = fuse.DT_File
		case stat_t.Mode&unix.S_IFDIR != 0:
			fuseEnts[i].Type = fuse.DT_Dir
		case stat_t.Mode&unix.S_IFBLK != 0:
			fuseEnts[i].Type = fuse.DT_Block
		case stat_t.Mode&unix.S_IFCHR != 0:
			fuseEnts[i].Type = fuse.DT_Char
		case stat_t.Mode&unix.S_IFIFO != 0:
			fuseEnts[i].Type = fuse.DT_FIFO
		default:
			fuseEnts[i].Type = fuse.DT_Unknown
		}
	}
	fmt.Println("readdir returning", len(fuseEnts), err)
	return fuseEnts, err
}

func (f *LocalFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if req.Size > 4096 * 16 {
		req.Size = 4096 * 16
	}
	resp.Data = make([]byte, 0, req.Size)
	n, err := unix.Pread(f.fd, resp.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Data = resp.Data[:n]
	return nil
}

func (f *LocalFile) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	f.ln.md.lock.Lock()
	delete(f.ln.md.openFiles, f.fd)
	f.ln.md.lock.Unlock()

	fmt.Println("close", f.fd, f.ln.Path)
	err := unix.Close(f.fd)
	var err2 error
	if f.osFile != nil {
		err2 = f.osFile.Close()
	}
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (f *LocalFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	n, err := unix.Pwrite(f.fd, req.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Size = n
	return nil
}