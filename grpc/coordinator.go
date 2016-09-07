package coordinator

import (
	"context"
	"os"
	"time"

	"bazil.org/fuse"
)

type CoordinatorServer interface {
	UserDirInfo(ctx context.Context, login string) (*LoginInfo, error)
	UserDirStat(ctx context.Context, login string) (*FileAttr, error)
	MyINode(ctx context.Context) uint64
}

type UserConnection interface {
	Access(ctx context.Context, path string, mode uint32) error
	Stat(ctx context.Context, path string) (*FileAttr, error)
	Getxattr(ctx context.Context, path string, attr string, size uint32, position uint32) ([]byte, error)
	Listxattr(ctx context.Context, path string, size uint32, position uint32) ([]byte, error)
	Open(ctx context.Context, path string, dir bool, flags AgnosticOpenFlags) (rflags fuse.OpenResponseFlags, fd uint64, err error)
	Readlink(ctx context.Context, path string) (string, error)
	LookupExists(ctx context.Context, path string) error

	ReadDir(ctx context.Context, fd uint64) ([]Dirent, error)
	ReadFrom(ctx context.Context, req *ReadRequest) ([]byte, error)
	Close(ctx context.Context, fd uint64) error
}

const (
	LookupNotFound = iota
	LookupIsFile
	LookupIsDir
)

type LoginInfo struct {
	Login     string
	Exists    bool
	WasOnline bool
}

type ReadRequest struct {
	FD        uint64
	Dir       bool
	Offset    int64
	Size      int
	FileFlags fuse.OpenFlags
}

type Dirent struct {
	Inode uint64
	Type  uint32
	Name  string
}

type FileAttr struct {
	INode     uint64
	Size      uint64
	Blocks    uint64
	Mtime     time.Time
	Ctime     time.Time
	BirthTime time.Time
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	BlockSize uint32
	Mode      os.FileMode
}
