// Copyright 2016 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

// +build !darwin

package libfuse

import "bazil.org/fuse"

func getPlatformSpecificMountOptions(dir string) ([]fuse.MountOption, error) {
	return []fuse.MountOption{}, nil
}

// GetPlatformSpecificMountOptionsForTest makes cross-platform tests work
func GetPlatformSpecificMountOptionsForTest() []fuse.MountOption {
	return []fuse.MountOption{}
}

func translatePlatformSpecificError(err error) error {
	return err
}
