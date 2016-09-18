package piccolo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Attr is the piccolo attributes for a file, i.e. verbatim, ignore, etc.
type Attr int

const (
	NONE     Attr = iota
	VERBATIM Attr = 1 << iota // Copy file w/o modification.
	INCLUDE                   // Add in header, footer and titlebar into HTML files.
	MAIN                      // The main index.html page goes in this directory.
	FEED                      // The index.atom feed goes in this directory.
	ARCHIVE                   // The archives goes in this directory.
	IGNORE                    // These files are not to be copied over.
	ROOT                      // The root of the publishing tree.
)

// Which attributes are non-cascading.
const NON_CASCADING = MAIN | FEED | ARCHIVE | ROOT

// filenames are the filenames that correspond to attributes.
var filenames = map[string]Attr{
	".verbatim":      VERBATIM,
	".include":       INCLUDE,
	".maintarget":    MAIN,
	".feedtarget":    FEED,
	".archivetarget": ARCHIVE,
	".ignore":        IGNORE,
	".root":          ROOT,
}

// Has returns true if the given Attr is present.
func (a Attr) Has(b Attr) bool {
	return a&b != 0
}

// Set sets the given Attr.
func (a *Attr) Set(b Attr) {
	*a |= b
}

// String representation of an Attr.
func (a Attr) String() string {
	s := []string{}
	for k, v := range filenames {
		if a.Has(v) {
			s = append(s, k)
		}
	}
	return "[" + strings.Join(s, ", ") + "]"
}

// DocSet keeps track of the attributes of all the directories and also tracks the
// important file locations.
type DocSet struct {
	// Cache of calculated arribute sets, indexed by path.
	cache map[string]Attr

	// The root of the tree.
	Root string

	// The directory where the main page goes.
	Main string

	// The directory where the archive pages go.
	Archive string

	// The directory where the Atom feed goes.
	Feed string
}

// URL tranforms a src path into a relative URL.
func (a DocSet) URL(path string) (string, error) {
	rel, err := filepath.Rel(a.Root, path)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(rel, ".html") {
		rel = rel[:len(rel)-5]
	}
	return "/" + rel, nil
}

// Dest tranforms a src path into a destination path.
func (a DocSet) Dest(path string) (string, error) {
	rel, err := filepath.Rel(a.Root, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(a.Root, "dst", rel), nil
}

// dirAttributes returns the attributes for a path.
func (a *DocSet) dirAttributes(path string) (Attr, error) {
	// If current dir has a .root
	matches, err := filepath.Glob(filepath.Join(path, ".*"))
	if err != nil {
		return NONE, err
	}
	var attr Attr
	for _, match := range matches {
		_, name := filepath.Split(match)
		if value, ok := filenames[name]; ok {
			attr.Set(value)
		}
	}
	return attr, nil
}

// merge returns the attributes for a child directory given the parents attributes.
func merge(parent, child Attr) Attr {
	res := child & NON_CASCADING
	if parent.Has(IGNORE) || child.Has(IGNORE) {
		res.Set(IGNORE)
		return res
	}
	if parent.Has(VERBATIM) {
		if child.Has(INCLUDE) {
			res.Set(INCLUDE)
		} else {
			res.Set(VERBATIM)
		}
	}
	if parent.Has(INCLUDE) {
		if child.Has(VERBATIM) {
			res.Set(VERBATIM)
		} else {
			res.Set(INCLUDE)
		}
	}
	return res
}

// Path returns the attributes for a path.
func (a *DocSet) Path(path string) (Attr, error) {
	if value, ok := a.cache[path]; ok {
		return value, nil
	}
	attr, err := a.dirAttributes(path)
	if err != nil {
		return NONE, err
	}
	if attr.Has(ROOT) {
		if a.Root != "" {
			return NONE, fmt.Errorf("Multiple .roots found: %s, %s\n", a.Root, path)
		}
		a.Root = path
	}
	if attr.Has(MAIN) {
		if a.Main != "" {
			return NONE, fmt.Errorf("Multiple .maintargets found: %s, %s\n", a.Main, path)
		}
		a.Main = path
	}
	if attr.Has(ARCHIVE) {
		if a.Archive != "" {
			return NONE, fmt.Errorf("Multiple .archivetargets found: %s, %s\n", a.Archive, path)
		}
		a.Archive = path
	}
	if attr.Has(FEED) {
		if a.Feed != "" {
			return NONE, fmt.Errorf("Multiple .feedtargets found: %s, %s\n", a.Feed, path)
		}
		a.Feed = path
	}

	if attr.Has(ROOT) {
		a.cache[path] = attr
		return attr, nil
	} else if path == "/" {
		return NONE, fmt.Errorf("Failed to find a .root.")
	}
	// Start trimming off path parts and call ourselves recursively.
	parentDir := filepath.Dir(path)
	parent, err := a.Path(parentDir)
	if err != nil {
		return NONE, err
	}

	// Combine attributes with the ones from our parent directory.
	newattr := merge(parent, attr)
	a.cache[path] = newattr
	return newattr, nil
}

// setKnownAttr makes our well-known directorys .ignore.
func (a *DocSet) setKnownAttr() {
	a.cache[filepath.Join(a.Root, "dst")] = IGNORE
	a.cache[filepath.Join(a.Root, "tmp")] = IGNORE
	a.cache[filepath.Join(a.Root, "tpl")] = IGNORE
	a.cache[filepath.Join(a.Root, "inc")] = IGNORE
	a.cache[filepath.Join(a.Root, ".git")] = IGNORE
}

// NewDocSet creates a new DocSet.
//
// path is a diretory path at or below the .root directory.
func NewDocSet(path string) (*DocSet, error) {
	a := &DocSet{cache: make(map[string]Attr)}
	_, err := a.Path(path)
	if err != nil {
		return nil, err
	} else {
		a.setKnownAttr()
	}
	return a, nil
}
