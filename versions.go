package main

import (
	"fmt"
	semver "github.com/Masterminds/semver/v3"
	"strconv"
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

func buildLabels(containerdVersion string, kubeletVersion string) (map[string]string, error) {
	containerdMajor, containerdMinor, err := getContainerdVersion(containerdVersion)
	if err != nil {
		return nil, err
	}
	kubeletMajor, kubeletMinor, err := getMajorMinor(kubeletVersion)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"mwam.com/containerd-major-version": strconv.FormatUint(containerdMajor, 10),
		"mwam.com/containerd-minor-version": strconv.FormatUint(containerdMinor, 10),
		"mwam.com/kublelet-major-version":   strconv.FormatUint(kubeletMajor, 10),
		"mwam.com/kublelet-minor-version":   strconv.FormatUint(kubeletMinor, 10),
	}, nil
}
