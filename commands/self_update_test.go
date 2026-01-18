package commands

import (
	"context"
	"errors"
	"testing"
)

// MockUpdater implements Updater interface for testing
type MockUpdater struct {
	LatestRelease  *Release
	Found          bool
	DetectErr      error
	UpdateErr      error
	UpdateCalled   bool
	DetectedSlug   string
	UpdatedPath    string
	UpdatedRelease *Release
}

func (m *MockUpdater) DetectLatest(ctx context.Context, slug string) (*Release, bool, error) {
	m.DetectedSlug = slug
	return m.LatestRelease, m.Found, m.DetectErr
}

func (m *MockUpdater) UpdateTo(ctx context.Context, release *Release, cmdPath string) error {
	m.UpdateCalled = true
	m.UpdatedRelease = release
	m.UpdatedPath = cmdPath
	return m.UpdateErr
}

func TestDoSelfUpdate(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		checkOnly      bool
		mock           *MockUpdater
		wantErr        bool
		errContains    string
		wantUpdate     bool
	}{
		{
			name:           "dev build rejected",
			currentVersion: "dev",
			mock:           &MockUpdater{},
			wantErr:        true,
			errContains:    "cannot update development build",
			wantUpdate:     false,
		},
		{
			name:           "detect error",
			currentVersion: "1.0.0",
			mock: &MockUpdater{
				DetectErr: errors.New("network error"),
			},
			wantErr:     true,
			errContains: "failed to check for updates",
			wantUpdate:  false,
		},
		{
			name:           "no release found",
			currentVersion: "1.0.0",
			mock: &MockUpdater{
				Found: false,
			},
			wantErr:    false,
			wantUpdate: false,
		},
		{
			name:           "already up to date",
			currentVersion: "2.0.0",
			mock: &MockUpdater{
				Found: true,
				LatestRelease: &Release{
					Version:   "1.0.0",
					AssetURL:  "https://example.com/asset.tar.gz",
					AssetName: "fogit_linux_amd64.tar.gz",
				},
			},
			wantErr:    false,
			wantUpdate: false,
		},
		{
			name:           "check only - does not update",
			currentVersion: "1.0.0",
			checkOnly:      true,
			mock: &MockUpdater{
				Found: true,
				LatestRelease: &Release{
					Version:   "2.0.0",
					AssetURL:  "https://example.com/asset.tar.gz",
					AssetName: "fogit_linux_amd64.tar.gz",
				},
			},
			wantErr:    false,
			wantUpdate: false,
		},
		{
			name:           "update available - performs update",
			currentVersion: "1.0.0",
			checkOnly:      false,
			mock: &MockUpdater{
				Found: true,
				LatestRelease: &Release{
					Version:   "2.0.0",
					AssetURL:  "https://example.com/asset.tar.gz",
					AssetName: "fogit_linux_amd64.tar.gz",
				},
			},
			wantErr:    false,
			wantUpdate: true,
		},
		{
			name:           "update fails",
			currentVersion: "1.0.0",
			checkOnly:      false,
			mock: &MockUpdater{
				Found: true,
				LatestRelease: &Release{
					Version:   "2.0.0",
					AssetURL:  "https://example.com/asset.tar.gz",
					AssetName: "fogit_linux_amd64.tar.gz",
				},
				UpdateErr: errors.New("permission denied"),
			},
			wantErr:     true,
			errContains: "failed to update",
			wantUpdate:  true, // Update was attempted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := doSelfUpdate(ctx, tt.currentVersion, tt.checkOnly, tt.mock)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.wantUpdate != tt.mock.UpdateCalled {
				t.Errorf("UpdateCalled = %v, want %v", tt.mock.UpdateCalled, tt.wantUpdate)
			}
		})
	}
}

func TestDoSelfUpdate_CorrectSlug(t *testing.T) {
	mock := &MockUpdater{
		Found: true,
		LatestRelease: &Release{
			Version: "1.0.0",
		},
	}

	_ = doSelfUpdate(context.Background(), "1.0.0", true, mock)

	if mock.DetectedSlug != "eg3r/fogit" {
		t.Errorf("DetectedSlug = %q, want %q", mock.DetectedSlug, "eg3r/fogit")
	}
}

func TestDoSelfUpdate_PassesReleaseToUpdate(t *testing.T) {
	expectedRelease := &Release{
		Version:   "2.0.0",
		AssetURL:  "https://github.com/eg3r/fogit/releases/download/v2.0.0/fogit_linux_amd64.tar.gz",
		AssetName: "fogit_linux_amd64.tar.gz",
	}

	mock := &MockUpdater{
		Found:         true,
		LatestRelease: expectedRelease,
	}

	_ = doSelfUpdate(context.Background(), "1.0.0", false, mock)

	if mock.UpdatedRelease == nil {
		t.Fatal("UpdatedRelease is nil")
	}
	if mock.UpdatedRelease.Version != expectedRelease.Version {
		t.Errorf("Version = %q, want %q", mock.UpdatedRelease.Version, expectedRelease.Version)
	}
	if mock.UpdatedRelease.AssetURL != expectedRelease.AssetURL {
		t.Errorf("AssetURL = %q, want %q", mock.UpdatedRelease.AssetURL, expectedRelease.AssetURL)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRelease_LessOrEqual(t *testing.T) {
	tests := []struct {
		releaseVersion string
		currentVersion string
		want           bool
	}{
		{"1.0.0", "1.0.0", true},      // equal
		{"1.0.0", "2.0.0", true},      // release older
		{"2.0.0", "1.0.0", false},     // release newer
		{"1.2.0", "1.3.0", true},      // minor older
		{"1.3.0", "1.2.0", false},     // minor newer
		{"1.2.3", "1.2.4", true},      // patch older
		{"1.2.4", "1.2.3", false},     // patch newer
		{"v1.0.0", "1.0.0", true},     // with v prefix
		{"1.0.0", "v1.0.0", true},     // current with v prefix
		{"v2.0.0", "v1.0.0", false},   // both with v prefix
		{"1.0.0-rc.1", "1.0.0", true}, // prerelease treated as 1.0.0
	}

	for _, tt := range tests {
		t.Run(tt.releaseVersion+"_vs_"+tt.currentVersion, func(t *testing.T) {
			r := &Release{Version: tt.releaseVersion}
			got := r.LessOrEqual(tt.currentVersion)
			if got != tt.want {
				t.Errorf("Release{%q}.LessOrEqual(%q) = %v, want %v",
					tt.releaseVersion, tt.currentVersion, got, tt.want)
			}
		})
	}
}
