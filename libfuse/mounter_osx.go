// Copyright 2016 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// +build darwin

package libfuse

import (
	"errors"

	"bazil.org/fuse"
)

var kbfusePath = fuse.OSXFUSEPaths{
	DevicePrefix: "/dev/kbfuse",
	Load:         "/Library/Filesystems/kbfuse.fs/Contents/Resources/load_kbfuse",
	Mount:        "/Library/Filesystems/kbfuse.fs/Contents/Resources/mount_kbfuse",
	DaemonVar:    "MOUNT_KBFUSE_DAEMON_PATH",
}

func getPlatformSpecificMountOptions(dir string) ([]fuse.MountOption, error) {
	options := []fuse.MountOption{}

	var locationOption fuse.MountOption
	locationOption = fuse.OSXFUSELocations(fuse.OSXFUSELocationV3)
	// allow keybase's copy of fuse if installed
	locationOption = fuse.OSXFUSELocations(kbfusePath)
	options = append(options, locationOption)

	// Volume name option is only used on OSX (ignored on other platforms).
	volName, err := volumeName(dir)
	if err != nil {
		return nil, err
	}

	options = append(options, fuse.VolumeName(volName))
	options = append(options, fuse.ExclCreate())

	return options, nil
}

// GetPlatformSpecificMountOptionsForTest makes cross-platform tests work
func GetPlatformSpecificMountOptionsForTest() []fuse.MountOption {
	// For now, test with either kbfuse or OSXFUSE for now.
	// TODO: Consider mandate testing with kbfuse?
	return []fuse.MountOption{
		fuse.OSXFUSELocations(kbfusePath, fuse.OSXFUSELocationV3),
		fuse.ExclCreate(),
	}
}

func translatePlatformSpecificError(err error) error {
	if err == fuse.ErrOSXFUSENotFound {
		return errors.New(
			"cannot locate kbfuse; either install the Keybase " +
				"app, or install OSXFUSE 3.x (3.2 " +
				"recommended) and pass in --use-system-fuse")
	}
	return err
}
