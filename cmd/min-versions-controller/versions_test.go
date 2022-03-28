package main

import (
	"reflect"
	"testing"
)

func Test_getContainerdVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		major   uint64
		minor   uint64
		wantErr bool
	}{
		{"happy case", "containerd://1.5.10", 1, 5, false},
		{"too short", "1.5.10", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := getContainerdVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContainerdVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if major != tt.major {
				t.Errorf("getContainerdVersion() major = %v, major %v", major, tt.major)
			}
			if minor != tt.minor {
				t.Errorf("getContainerdVersion() minor = %v, major %v", minor, tt.minor)
			}
		})
	}
}

func Test_getMajorMinor(t *testing.T) {
	tests := []struct {
		name    string
		version string
		major   uint64
		minor   uint64
		wantErr bool
	}{
		{"happy case", "v1.5.10", 1, 5, false},
		{"wrong", "some_words.1.5.10", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := getMajorMinor(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContainerdVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if major != tt.major {
				t.Errorf("getContainerdVersion() major = %v, major %v", major, tt.major)
			}
			if minor != tt.minor {
				t.Errorf("getContainerdVersion() minor = %v, major %v", minor, tt.minor)
			}
		})
	}
}

func Test_buildTags(t *testing.T) {
	type args struct {
		containerdVersion string
		kubeletVersion    string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			"happy case",
			args{"containerd://1.5.10", "v1.22.7"},
			map[string]string{
				"mwam.com/containerd-major-version": "1",
				"mwam.com/containerd-minor-version": "5",
				"mwam.com/kubelet-major-version":    "1",
				"mwam.com/kubelet-minor-version":    "22",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildNodeLabels(tt.args.containerdVersion, tt.args.kubeletVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildNodeLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildNodeLabels() got = %v, want %v", got, tt.want)
			}
		})
	}
}
