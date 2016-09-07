// +build darwin,cgo

package fscore

import (
	"unsafe"

	"bazil.org/fuse"
	"golang.org/x/net/context"
)

/*
#include <sys/xattr.h>
#include <stdlib.h>
#include <errno.h>

ssize_t bridge_getxattr(char *path, char *attrName, void **buf,
	size_t size, u_int32_t position)
{
	ssize_t size_read;

	errno = 0;
	if (size == 0)
		*buf = 0;
	else
		*buf = malloc(size);
	size_read = getxattr(path, attrName, *buf, size, position, XATTR_NOFOLLOW);
	free(path);
	free(attrName);
	return (size_read);
}

ssize_t	bridge_listxattr(char *path, void **buf, size_t size)
{
	ssize_t size_read;

	errno = 0;
	if (size == 0)
		*buf = 0;
	else
		*buf = malloc(size);
	size_read = listxattr(path, *buf, size, XATTR_NOFOLLOW);
	free(path);
	return (size_read);
}

int	bridge_removexattr(char *path, char *name)
{
	int eret;

	errno = 0;
	eret = removexattr(path, name, XATTR_NOFOLLOW);
	free(path);
	free(name);
	return (eret);
}

int	bridge_setxattr(char *path, char *name, void *value, size_t size, u_int32_t position, int options)
{
	int eret;

	errno = 0;
	eret = setxattr(path, name, value, size, position, options | XATTR_NOFOLLOW);
	free(path);
	free(name);
	free(value);
	return (eret);
}

*/
import "C"

func (d *LocalNode) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	var bufAddr unsafe.Pointer
	var bufLen C.ssize_t

	bufLen, err := C.bridge_getxattr(C.CString(d.FullPath()), C.CString(req.Name), &bufAddr, C.size_t(req.Size), req.Position)
	if bufLen == -1 {
		return err
	}
	if req.Size == 0 {
		resp.Xattr = make([]byte, int(bufLen))
	} else {
		resp.Xattr = C.GoBytes(bufAddr, C.int(bufLen))
		C.free(bufAddr)
	}
	return nil
}

func (d *LocalNode) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	var bufAddr unsafe.Pointer
	var bufLen C.ssize_t

	bufLen, err := C.bridge_listxattr(C.CString(d.FullPath()), &bufAddr, C.size_t(req.Size))
	if bufLen == -1 {
		return err
	}
	if req.Size == 0 {
		resp.Xattr = make([]byte, int(bufLen))
	} else {
		resp.Xattr = C.GoBytes(bufAddr, C.int(bufLen))
		C.free(bufAddr)
	}
	return nil
}

func (d *LocalNode) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	_, err := C.bridge_removexattr(C.CString(d.FullPath()), C.CString(req.Name))
	return err
}

func (d *LocalNode) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	_, err := C.bridge_setxattr(C.CString(d.FullPath()), C.CString(req.Name), C.CBytes(req.Xattr), C.size_t(len(req.Xattr)), req.Position, C.int(req.Flags))
	return err
}