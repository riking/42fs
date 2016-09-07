// build +darwin
package fscore

import (
	"os"
	"syscall"

	"time"

	"bazil.org/fuse"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

func (d *LocalNode) Attr(ctx context.Context, a *fuse.Attr) error {
	var stat_t unix.Stat_t
	err := unix.Lstat(d.FullPath(), &stat_t)
	if err != nil {
		return fuse.Errno(err.(syscall.Errno))
	}

	a.Inode = stat_t.Ino
	a.Size = uint64(stat_t.Size)
	a.Blocks = uint64(stat_t.Blocks)
	a.Atime = time.Unix(stat_t.Atimespec.Unix())
	a.Mtime = time.Unix(stat_t.Mtimespec.Unix())
	a.Ctime = time.Unix(stat_t.Ctimespec.Unix())
	a.Crtime = time.Unix(stat_t.Birthtimespec.Unix())
	a.Mode = os.FileMode(stat_t.Mode)
	a.Nlink = uint32(stat_t.Nlink)
	a.Uid = uint32(stat_t.Uid)
	a.Gid = uint32(stat_t.Gid)
	a.Rdev = uint32(stat_t.Rdev)
	a.Flags = uint32(stat_t.Flags)
	a.BlockSize = uint32(stat_t.Blksize)
	return nil
}

func (d *LocalNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	fd, err := unix.Open(d.FullPath(), unix.O_NOFOLLOW | unix.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	return unix.Fsync(fd)
}

func (d *LocalNode) setattrPlatform(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	var err, finalErr error
	addErr := func(e error) {
		if finalErr == nil {
			finalErr = e
		}
	}

	if req.Valid.Flags() {
		err = unix.Chflags(d.FullPath(), int(req.Flags))
		addErr(err)
	}
	if req.Valid.Bkuptime() || req.Valid.Chgtime() || req.Valid.Crtime() {
		addErr(fuse.ENOTSUP)
	}
	return finalErr
}
