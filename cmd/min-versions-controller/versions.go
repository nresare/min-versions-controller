package main

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
)

const (
	ContainerdMajorKey = "mwam.com/containerd-major-version"
	ContainerdMinorKey = "mwam.com/containerd-minor-version"
	KubeletMajorKey    = "mwam.com/kubelet-major-version"
	KubeletMinorKey    = "mwam.com/kubelet-minor-version"
)

func getContainerdVersion(version string) (uint64, uint64, error) {
	const PREFIX = "containerd://"
	if len(version) < len(PREFIX) {
		return 0, 0, fmt.Errorf("version string '%s' does not match expected format 'conainerd://1.2.3", version)
	}
	version = version[len(PREFIX):]
	major, minor, err := getMajorMinor(version)
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}

func getMajorMinor(version string) (uint64, uint64, error) {
	ver, err := semver.NewVersion(version)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse version '%s': %w", version, err)
	}
	return ver.Major(), ver.Minor(), nil
}
