package coordinator

import (
	"bazil.org/fuse"
	"golang.org/x/sys/unix"
	"syscall"
)

type AgnosticOpenFlags uint32

type flagMapping struct {
	Agnostic AgnosticOpenFlags
	Sys      fuse.OpenFlags
}

var flagMap = []flagMapping{
	{0x8, syscall.O_APPEND},
	{0x10, syscall.O_CREAT},
	{0x20, syscall.O_DIRECTORY},
	{0x40, syscall.O_EXCL},
	{0x80, syscall.O_NONBLOCK},
	{0x100, syscall.O_SYNC},
	{0x200, syscall.O_TRUNC},
}

func ToAgnostic(sys fuse.OpenFlags) AgnosticOpenFlags {
	var ag AgnosticOpenFlags
	ag = AgnosticOpenFlags(sys & unix.O_ACCMODE)
	for _, v := range flagMap {
		if (sys & v.Sys != 0) {
			ag |= v.Agnostic
		}
	}
	return ag
}

func (ag AgnosticOpenFlags) ToSys() fuse.OpenFlags {
	var sys fuse.OpenFlags
	sys = fuse.OpenFlags(ag & unix.O_ACCMODE)
	for _, v := range flagMap {
		if (ag & v.Agnostic != 0) {
			sys |= v.Sys
		}
	}
	return sys
}