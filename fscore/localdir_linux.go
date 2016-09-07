//build +linux
package fscore

import (
	"syscall"
	"time"

	"bazil.org/fuse"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"fmt"
)

func (d *LocalNode) Attr(ctx context.Context, a *fuse.Attr) error {
	var stat_t unix.Stat_t
	fmt.Println("stat", d.Path)
	err := unix.Lstat(d.FullPath(), &stat_t)
	if err != nil {
		return fuse.Errno(err.(syscall.Errno))
	}

	a.Inode = stat_t.Ino
	a.Size = uint64(stat_t.Size)
	a.Blocks = uint64(stat_t.Blocks)
	a.Atime = time.Unix(stat_t.Atim.Unix())
	a.Mtime = time.Unix(stat_t.Mtim.Unix())
	a.Ctime = time.Unix(stat_t.Ctim.Unix())
	//a.Crtime = time.Unix(stat_t.Birthtimespec.Unix())
	a.Mode = fileMode(stat_t.Mode)
	a.Nlink = uint32(stat_t.Nlink)
	a.Uid = uint32(stat_t.Uid)
	a.Gid = uint32(stat_t.Gid)
	a.Rdev = uint32(stat_t.Rdev)
	//a.Flags = uint32(stat_t.Flags)
	a.BlockSize = uint32(stat_t.Blksize)
	return nil
}

func (d *LocalNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	fd, err := unix.Open(d.FullPath(), unix.O_NOFOLLOW | unix.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	if req.Flags & 1 != 0 {
		return unix.Fdatasync(fd)
	} else {
		return unix.Fsync(fd)
	}
}

func (d *LocalNode) setattrPlatform(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return nil
}
