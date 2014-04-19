package piccolo

import (
	"code.google.com/p/go.net/html"
	"fmt"
	"os"
	"time"
)

const format = "2006-01-02T15:04:05-07:00"
const format_no_tz = "2006-01-02T15:04:05"

// FileInfo contains information about each file that has include processing
// done on it.
type FileInfo struct {
	// Filesystem path to the source file.
	Path string

	// Parsed HTML of the file.
	Node *html.Node

	// The extracted title.
	Title string

	// Time the source file was created.
	Created time.Time

	// Time the source file was last updated.
	Updated time.Time
}

// Body returns the parsed html.Node's in the body.
func (f FileInfo) Body() []*html.Node {
	body := make([]*html.Node, 0)

	var found func(*html.Node)
	found = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				body = append(body, c)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			found(c)
		}
	}
	found(f.Node)
	return body
}

func getAttrByName(node *html.Node, name string) (string, error) {
	for _, a := range node.Attr {
		if a.Key == name {
			return a.Val, nil
		}
	}
	return "", fmt.Errorf("Attribute %s not found.", name)
}

// CreationDate returns the time an HTML document was created.
//
// It also returns a FileInfo for the document, with the time added in the
// header if it was missing. The bool returned is true the meta creation
// element has been added to the header.
func CreationDate(path string) (*FileInfo, bool, error) {
	title := ""
	f, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, false, err
	}

	doc, err := html.Parse(f)
	if err != nil {
		return nil, false, err
	}
	hasMeta := false
	var head *html.Node
	var found func(*html.Node)
	var created time.Time
	found = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "head" {
			head = n
		}
		if n.Type == html.ElementNode && n.Data == "title" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					title = title + c.Data
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			name, err := getAttrByName(n, "name")
			if err == nil {
				value, err := getAttrByName(n, "value")
				if err == nil && name == "created" {
					created, err = time.Parse(format, value)
					if err != nil {
						created, err = time.Parse(format_no_tz, value)
						if err == nil {
							hasMeta = true
						}
					} else {
						hasMeta = true
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			found(c)
		}
	}
	found(doc)

	if !hasMeta {
		now := time.Now()
		meta := &html.Node{
			Type: html.ElementNode,
			Data: "meta", Attr: []html.Attribute{
				{Key: "value", Val: now.Format(format)},
				{Key: "name", Val: "created"},
			}}
		head.AppendChild(meta)
		created = now
	}
	fi := &FileInfo{
		Path:    path,
		Node:    doc,
		Title:   title,
		Created: created,
		Updated: stat.ModTime(),
	}
	return fi, !hasMeta, nil
}

// CreationDateSaved gets the creation date of an HTML file, and also writes
// that HTML file back in place with an updated meta element with the
// creation time if that information doesn't already exist.
func CreationDateSaved(path string) (*FileInfo, error) {
	fileinfo, update, err := CreationDate(path)
	if err != nil {
		return nil, err
	}
	if update {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		err = html.Render(f, fileinfo.Node)
		if err != nil {
			return nil, err
		}
	}
	return fileinfo, err
}
