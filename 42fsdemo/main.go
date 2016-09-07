package main

import (
	"fmt"
	"os"

	"github.com/riking/42fs/libfuse"

	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/riking/42fs/fscore"
)

func fatalErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	mounter := libfuse.NewForceMounter("../test/fusetest",
		fuse.FSName("42fs"),
		fuse.VolumeName("42fs"),
		fuse.NoAppleDouble(),
		//fuse.DefaultPermissions(),
	)
	conn, err := mounter.Mount()
	if err != nil {
		log.Fatal(err)
	}
	fs42 := fscore.NewFS42(nil, "kyork", "/home/kane/public")
	err = fs.Serve(conn, fs42)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("waiting for Ready...")
	<-conn.Ready
	if conn.MountError != nil {
		log.Fatal(conn.MountError)
	}
	defer mounter.Unmount()
	fmt.Println("protocol:", conn.Protocol())

	mounter.Unmount()
}
