// +build darwin,!cgo

package fscore

import (
	"bazil.org/fuse"
	"golang.org/x/net/context"
)

func (d *LocalNode) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return fuse.ENOTSUP
}

func (d *LocalNode) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	return fuse.ENOTSUP
}

func (d *LocalNode) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	return fuse.ENOTSUP
}

func (d *LocalNode) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	return fuse.ENOTSUP
}