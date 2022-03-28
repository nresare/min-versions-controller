package main

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	v1 "k8s.io/api/core/v1"
	"strconv"
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

func buildNodeLabels(containerdVersion string, kubeletVersion string) (map[string]string, error) {
	containerdMajor, containerdMinor, err := getContainerdVersion(containerdVersion)
	if err != nil {
		return nil, err
	}
	kubeletMajor, kubeletMinor, err := getMajorMinor(kubeletVersion)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		ContainerdMajorKey: strconv.FormatUint(containerdMajor, 10),
		ContainerdMinorKey: strconv.FormatUint(containerdMinor, 10),
		KubeletMajorKey:    strconv.FormatUint(kubeletMajor, 10),
		KubeletMinorKey:    strconv.FormatUint(kubeletMinor, 10),
	}, nil
}

func buildGreaterThanTerm(key string, value uint64) v1.NodeSelectorTerm {
	return v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{{
			key,
			v1.NodeSelectorOpGt,
			[]string{strconv.FormatUint(value, 10)},
		}},
	}
}

func buildNodeSelector(minContainerdVersion string, minKubeletVersion string) (*v1.NodeSelector, error) {
	nodeSelectorTerm := make([]v1.NodeSelectorTerm, 0)
	if minContainerdVersion != "" {
		containerdMajor, containerdMinor, err := getMajorMinor(minContainerdVersion)
		if err != nil {
			return nil, err
		}
		nodeSelectorTerm = append(nodeSelectorTerm, buildGreaterThanTerm(ContainerdMajorKey, containerdMajor-1))
		nodeSelectorTerm = append(nodeSelectorTerm, buildGreaterThanTerm(ContainerdMinorKey, containerdMinor-1))
	}

	if minKubeletVersion != "" {
		kubeletMajor, kubeletMinor, err := getMajorMinor(minKubeletVersion)
		if err != nil {
			return nil, err
		}
		nodeSelectorTerm = append(nodeSelectorTerm, buildGreaterThanTerm(KubeletMajorKey, kubeletMajor-1))
		nodeSelectorTerm = append(nodeSelectorTerm, buildGreaterThanTerm(KubeletMinorKey, kubeletMinor-1))

	}
	return &v1.NodeSelector{
		NodeSelectorTerms: nodeSelectorTerm,
	}, nil

}
