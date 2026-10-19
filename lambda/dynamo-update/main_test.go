package main

import (
	"testing"
)

func TestBuildKeywords(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []string
	}{
		{
			name:     "underscore to hyphen and uppercase",
			filename: "hoge_123",
			want:     []string{"HOGE-123", "HOGE123", "HOGE-123"},
		},
		{
			name:     "already uppercase with hyphen",
			filename: "HOGE-123",
			want:     []string{"HOGE-123", "HOGE123"},
		},
		{
			name:     "alphanumeric only generates hyphen variant",
			filename: "HOGE123",
			want:     []string{"HOGE123", "HOGE-123"},
		},
		{
			name:     "all three variants differ",
			filename: "ab_cd-123",
			want:     []string{"AB-CD-123", "ABCD123", "ABCD-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKeywords(tt.filename)
			// deduplicated expected
			seen := map[string]bool{}
			want := []string{}
			for _, w := range tt.want {
				if !seen[w] {
					seen[w] = true
					want = append(want, w)
				}
			}
			if len(got) != len(want) {
				t.Errorf("buildKeywords(%q) len = %d, want %d; got %v", tt.filename, len(got), len(want), got)
				return
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("buildKeywords(%q)[%d] = %q, want %q", tt.filename, i, got[i], want[i])
				}
			}
		})
	}
}

func TestParseS3Path(t *testing.T) {
	tests := []struct {
		name       string
		s3Path     string
		wantBucket string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "with prefix",
			s3Path:     "s3://my-bucket/path/to/dir",
			wantBucket: "my-bucket",
			wantPrefix: "path/to/dir",
		},
		{
			name:       "bucket only",
			s3Path:     "s3://my-bucket",
			wantBucket: "my-bucket",
			wantPrefix: "",
		},
		{
			name:    "invalid path",
			s3Path:  "s3://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, prefix, err := parseS3Path(tt.s3Path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseS3Path(%q) error = %v, wantErr %v", tt.s3Path, err, tt.wantErr)
				return
			}
			if bucket != tt.wantBucket {
				t.Errorf("parseS3Path(%q) bucket = %q, want %q", tt.s3Path, bucket, tt.wantBucket)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("parseS3Path(%q) prefix = %q, want %q", tt.s3Path, prefix, tt.wantPrefix)
			}
		})
	}
}
