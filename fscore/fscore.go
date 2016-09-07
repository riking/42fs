package fscore

import (
	"os"

	fgrpc "github.com/riking/42fs/grpc"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
	"fmt"
)

const (
	INodeRootDir uint64 = 1
	INodeREADME = 2
	INodeReserved3 = 3
	INodeReserved4 = 4
	INodeReserved5 = 5
)

type FS42 struct {
	coordCur   fgrpc.CoordinatorServer
	myLogin    string
	myRealPath string
	root       RootDir
	local      *LocalDir
}

func NewFS42(coord fgrpc.CoordinatorServer, myLogin string, myDir string) *FS42 {
	fs42 := &FS42{
		coordCur:      coord,
		myLogin:    myLogin,
		myRealPath: myDir,
	}
	fs42.root = RootDir{fs42: fs42}
	fs42.local = NewLocalDir(fs42, myDir)
	return fs42
}

func (fs42 *FS42) Root() (fs.Node, error) {
	return fs42.root, nil
}

func (fs42 *FS42) WhoAmI() string {
	return fs42.myLogin
}

func (fs42 *FS42) coord() fgrpc.CoordinatorServer {
	return nil
}

type RootDir struct {
	fs42 *FS42
}

func (d RootDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = INodeRootDir
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d RootDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println(d)
	if name == "README" {
		return ReadmeFile{}, nil
	} else if name == d.fs42.WhoAmI() {
		return d.fs42.local, nil
	} else {
		return nil, fuse.ENOENT // TODO
		info, err := d.fs42.coord().UserDirInfo(ctx, name)
		if err != nil {
			return nil, err
		}
		if !info.Exists {
			return nil, fuse.ENOENT
		}
		ud := NewUserDir(d.fs42, info)
		return &ud.RemoteNode, nil
	}
}

var rootEntries = []fuse.Dirent{
	{Inode: INodeREADME, Name: "README", Type: fuse.DT_File},
}

func (r RootDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := make([]fuse.Dirent, len(rootEntries) + 1)
	copy(entries, rootEntries)
	entries[len(rootEntries)] = fuse.Dirent{
		//Inode: r.fs42.local.Inode(ctx),
		Name:  r.fs42.WhoAmI(),
		Type:  fuse.DT_Dir,
	}
	return entries, nil
}

type ReadmeFile struct{}

const readmeContent = `
Welcome to fs42!

Here you can place files so they can be accessed by other 42 users.
In order to browse someone's fs42 directory, you must both be logged in.

`

func (ReadmeFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = INodeREADME
	a.Mode = 0444
	a.Size = uint64(len(readmeContent))
	return nil
}

func (ReadmeFile) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(readmeContent), nil
}
