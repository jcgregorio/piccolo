package piccolo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTransform(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v\n", err)
	}
	testCases := []struct {
		Filename string
		IsNew    bool
		Title    string
	}{
		{"transform.html", false, " Test date supplied. "},
		{"transform2.html", true, " No date supplied"},
	}
	for _, tc := range testCases {
		path := filepath.Join(cwd, "tests", "src", tc.Filename)
		fi, isNew, err := CreationDate(path)

		if got, want := fi.Title, tc.Title; got != want {
			t.Errorf("Title wrong for %s, Got %v, Want %v\n", tc.Filename, got, want)
		}
		if isNew != tc.IsNew {
			t.Errorf("Metadata expectations wrong for %s, Want %v, Got %v\n", tc.Filename, tc.IsNew, isNew)
		}
		if err != nil {
			t.Errorf("Error getting creation date: %v in %s\n", err, tc.Filename)
		}
		expected := time.Date(2011, time.January, 1, 0, 0, 0, 0, time.UTC)
		if fi.Created.Before(expected) {
			t.Errorf("Unexpected date: %v in %s\n", fi.Created, tc.Filename)
		}
	}
}
