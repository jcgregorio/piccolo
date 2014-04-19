package piccolo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttrbutes(t *testing.T) {
	testCases := []struct {
		Dir  string
		Want Attr
	}{
		{"test1", VERBATIM | ROOT},
		{"test1/a", INCLUDE},
		{"test1/a/b", VERBATIM},
		{"test1/c", IGNORE},
		{"test1/feed", FEED | VERBATIM},
		{"test1/archives", ARCHIVE | VERBATIM},
		{"test1/inc", IGNORE},
		{"test1/tpl", IGNORE},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v\n", err)
	}
	testDir := filepath.Join(cwd, "tests", "src")
	for _, tc := range testCases {
		dir := filepath.Join(testDir, tc.Dir)
		a, err := NewDocSet(dir)
		if err != nil {
			t.Fatalf("Failed to build DocSet: %v\n", err)
		}
		attr, err := a.Path(dir)
		if err != nil {
			t.Fatalf("Failed to get attributes: %v\n", err)
		}
		if attr != tc.Want {
			t.Fatalf("Failed to match for %s. Got %v, Want %v\n", tc.Dir, attr, tc.Want)
		}
	}
}

func TestKnownPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v\n", err)
	}
	testDir := filepath.Join(cwd, "tests", "src", "test1")
	a, err := NewDocSet(testDir)
	if err != nil {
		t.Fatalf("Failed to build DocSet: %v\n", err)
	}
	walker := func(path string, info os.FileInfo, err error) error {
		_, err = a.Path(path)
		if err != nil {
			return err
		}
		return nil
	}
	err = filepath.Walk(testDir, walker)
	if err != nil {
		t.Fatalf("Failed to walk the tree: %v\n", err)
	}
	if a.Root != testDir {
		t.Fatalf("Failed to find the .root: Got %s Want %s\n", a.Root, testDir)
	}
	archiveDir := filepath.Join(testDir, "archives")
	if a.Archive != archiveDir {
		t.Fatalf("Failed to find the .root: Got %s Want %s\n", a.Archive, archiveDir)
	}
	feedDir := filepath.Join(testDir, "feed")
	if a.Feed != feedDir {
		t.Fatalf("Failed to find the .feed: Got %s Want %s\n", a.Feed, feedDir)
	}
	if a.Main != "" {
		t.Fatalf("Shouldn't have found .main: Got %s\n", a.Main)
	}
}

func TestFailures(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v\n", err)
	}
	testDir := filepath.Join(cwd, "tests", "src", "test2")
	a, err := NewDocSet(testDir)
	if err != nil {
		t.Fatalf("Failed to build DocSet: %v\n", err)
	}
	walker := func(path string, info os.FileInfo, err error) error {
		_, err = a.Path(path)
		if err != nil {
			return err
		}
		return nil
	}
	err = filepath.Walk(testDir, walker)
	if err == nil {
		t.Fatalf("Should have failed on duplicate archives.\n")
	}
}
