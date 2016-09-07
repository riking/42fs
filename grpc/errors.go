package coordinator

import (
	"syscall"

	"bazil.org/fuse"
)

type FS42GrpcErr struct {
	ErrNum int32  `json:"en"`
	Msg    string `json:"m"`
}

func (e FS42GrpcErr) Error() string {
	if e.ErrNum != 0 {
		eno := fuse.Errno(syscall.Errno(e.ErrNum))
		return eno.Error()
	} else {
		return e.Msg
	}
}

func (e FS42GrpcErr) Errno() fuse.Errno {
	return fuse.Errno(syscall.Errno(e.ErrNum))
}
