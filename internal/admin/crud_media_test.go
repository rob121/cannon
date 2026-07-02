package admin

import "testing"

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{512, "512 B"},
		{2048, "2 KB"},
		{1536, "1.5 KB"},
		{1048576, "1 MB"},
	}
	for _, tc := range tests {
		if got := formatFileSize(tc.size); got != tc.want {
			t.Fatalf("formatFileSize(%d) = %q, want %q", tc.size, got, tc.want)
		}
	}
}

func TestIsImageMIME(t *testing.T) {
	if !isImageMIME("image/png") {
		t.Fatal("image/png should be image")
	}
	if isImageMIME("application/pdf") {
		t.Fatal("pdf should not be image")
	}
}
