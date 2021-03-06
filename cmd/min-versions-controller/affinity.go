package main

import (
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"strconv"
)

func buildGreaterThanRequirement(key string, value uint64) corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      key,
		Operator: corev1.NodeSelectorOpGt,
		Values:   []string{strconv.FormatUint(value, 10)},
	}
}

func buildNodeSelector(minContainerdVersion string, minKubeletVersion string) (*corev1.NodeSelector, error) {
	matchExpressions := make([]corev1.NodeSelectorRequirement, 0)
	if minContainerdVersion != "" {
		containerdMajor, containerdMinor, err := getMajorMinor(minContainerdVersion)
		if err != nil {
			return nil, err
		}
		matchExpressions = append(matchExpressions, buildGreaterThanRequirement(ContainerdMajorKey, containerdMajor-1))
		matchExpressions = append(matchExpressions, buildGreaterThanRequirement(ContainerdMinorKey, containerdMinor-1))
	}

	if minKubeletVersion != "" {
		kubeletMajor, kubeletMinor, err := getMajorMinor(minKubeletVersion)
		if err != nil {
			return nil, err
		}
		matchExpressions = append(matchExpressions, buildGreaterThanRequirement(KubeletMajorKey, kubeletMajor-1))
		matchExpressions = append(matchExpressions, buildGreaterThanRequirement(KubeletMinorKey, kubeletMinor-1))

	}
	return &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: matchExpressions}},
	}, nil

}

func getRequiredAffinity(pod *corev1.Pod) *corev1.NodeSelector {
	affinity := pod.Spec.Affinity
	if affinity == nil {
		return nil
	}
	nodeAffinity := affinity.NodeAffinity
	if nodeAffinity == nil {
		return nil
	}
	return nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
}

func in(value corev1.NodeSelectorTerm, slice []corev1.NodeSelectorTerm) bool {
	for _, item := range slice {
		if reflect.DeepEqual(item, value) {
			return true
		}
	}
	return false
}

func mergeRequired(first *corev1.NodeSelector, second *corev1.NodeSelector) *corev1.NodeSelector {
	var finalTerms []corev1.NodeSelectorTerm
	if first != nil {
		copy(finalTerms, first.NodeSelectorTerms)
	}

	for _, term := range second.NodeSelectorTerms {
		if !in(term, finalTerms) {
			finalTerms = append(finalTerms, term)
		}
	}
	return &corev1.NodeSelector{NodeSelectorTerms: finalTerms}
}

func replaceAffinityNodeSelector(pod *corev1.Pod, nodeSelector *corev1.NodeSelector) {
	affinity := pod.Spec.Affinity
	if affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector,
			},
		}
		return
	}
	nodeAffinity := affinity.NodeAffinity
	if nodeAffinity == nil {
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector,
		}
		return
	}
	nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodeSelector
}
