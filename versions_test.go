package main

import "testing"

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
