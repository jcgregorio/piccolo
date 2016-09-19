package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/jcgregorio/piccolo/piccolo"
	"golang.org/x/net/html"
)

const (
	SITE_TITLE = "BitWorking"
	DOMAIN     = "https://bitworking.org/"
	FEED_LEN   = 4
)

var shortMonths = [...]string{
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	os.Exit(1)
}

// ShortMonth returns the short English name of the month ("Jan", "Feb", ...).
func ShortMonth(m time.Month) string { return shortMonths[m-1] }

type datediffer func(time.Time) string

// datediff returns a function that formats the archive entries correctly.
//
// The returned function is a closure that keeps track of the last time.Time it
// saw which it needs to do the formatting correctly.
func datediff() datediffer {
	var last time.Time

	return func(t time.Time) string {
		r := ""
		if t.After(last) {
			r = fmt.Sprintf("foo %#v", t)
		}
		// If years differ, emit year, month, day
		if t.Year() != last.Year() {
			r = fmt.Sprintf("<i><b>%d</b></i></td><td></td></tr>\n    <tr><td><b>%s</b></td><td></td></tr>\n    <tr><td> %d", t.Year(), ShortMonth(t.Month()), t.Day())
		} else if t.Month() != last.Month() {
			r = fmt.Sprintf("<b>%s</b></td><td></td></tr>\n   <tr><td> %d", ShortMonth(t.Month()), t.Day())
		} else {
			r = fmt.Sprintf("%d", t.Day())
		}
		last = t
		return r
	}
}

// trunc10 formats a time to just the year, month and day in ISO format.
func trunc10(t time.Time) string {
	return t.Format("2006-01-02")
}

// rfc339 formats a time in RFC3339 format.
func rfc3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// Templates contains all the parsed templates.
type Templates struct {
	IndexHTML   *template.Template
	IndexAtom   *template.Template
	ArchiveHTML *template.Template
	EntryHTML   *template.Template
}

func loadTemplate(d *piccolo.DocSet, name string) *template.Template {
	funcMap := template.FuncMap{
		"datediff": datediff(),
		"trunc10":  trunc10,
		"rfc3339":  rfc3339,
	}

	fullname := filepath.Join(d.Root, "tpl", name)
	return template.Must(template.New(name).Funcs(funcMap).ParseFiles(fullname))
}

func loadTemplates(d *piccolo.DocSet) *Templates {
	return &Templates{
		IndexHTML:   loadTemplate(d, "index.html"),
		IndexAtom:   loadTemplate(d, "index.atom"),
		ArchiveHTML: loadTemplate(d, "archive.html"),
		EntryHTML:   loadTemplate(d, "entry.html"),
	}
}

// Expand expands the template with the given data.
func Expand(d *piccolo.DocSet, t *template.Template, data interface{}, path string) error {
	dst, err := d.Dest(path)
	if err != nil {
		return err
	}
	dstDir, _ := filepath.Split(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	t.Execute(out, data)
	return nil
}

// SimpleInclude loads the include file given the docset d.
//
func SimpleInclude(d *piccolo.DocSet, filename string) (string, time.Time, error) {
	fullname := filepath.Join(d.Root, filename)

	f, err := os.Open(fullname)
	if err != nil {
		return "", time.Time{}, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", time.Time{}, err
	}
	t := stat.ModTime()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", time.Time{}, err
	}
	return string(b), t, nil
}

// Include loads the include file given the docset d.
//
// Returns the extracted HTML and the time the file was last modified.
func Include(d *piccolo.DocSet, filename, element string) (string, time.Time, error) {
	fullname := filepath.Join(d.Root, "inc", filename)

	f, err := os.Open(fullname)
	if err != nil {
		return "", time.Time{}, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", time.Time{}, err
	}
	t := stat.ModTime()

	doc, err := html.Parse(f)
	if err != nil {
		return "", time.Time{}, err
	}

	var found func(*html.Node)
	children := []*html.Node{}
	found = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == element {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				children = append(children, c)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			found(c)
		}
	}
	found(doc)
	return StrFromNodes(children), t, nil
}

// Newest returns the most recent of all the times passed in.
func Newest(times ...time.Time) time.Time {
	newest := times[0]
	for _, t := range times {
		if t.After(newest) {
			newest = t
		}
	}
	return newest
}

// StrFromNodes returns the string of the rendered html.Nodes.
func StrFromNodes(nodes []*html.Node) string {
	buf := bytes.NewBuffer([]byte{})
	for _, h := range nodes {
		html.Render(buf, h)
	}
	return buf.String()
}

// Entry represents a single blog entry.
type Entry struct {
	// Path is the source file path.
	Path string

	// Title is the title of the entry.
	Title string

	// URL is the relative URL of the file.
	URL string

	// Created is the created time.
	Created time.Time

	// Upated is the updated time.
	Updated time.Time

	// Body is the string representation of the body element, w/o
	// the <body> tags.
	Body string
}

// EntryByCreated is a type that allows sorting Entries by their created time.
type EntryByCreated []*Entry

func (s EntryByCreated) Len() int           { return len(s) }
func (s EntryByCreated) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s EntryByCreated) Less(i, j int) bool { return s[i].Created.After(s[j].Created) }

// TemplateData is the data used for expanding the index and archive (html and atom) templates.
type TemplateData struct {
	// Domain is the domain name the site will be served from.
	Domain string

	SiteTitle string
	Header    string
	InlineCSS string
	Titlebar  string
	Footer    string
	Entries   []*Entry

	// Most recent time anything on the site was updated.
	Updated time.Time
}

func modifiedTime(path string) time.Time {
	mod := time.Time{}
	if stat, err := os.Stat(path); err == nil {
		mod = stat.ModTime()
	}
	return mod
}

func incMust(s string, t time.Time, err error) (string, time.Time) {
	if err != nil {
		log.Fatalf("Error loading header: %v\n", err)
	}
	return s, t
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get cwd: %v\n", err)
	}
	d, err := piccolo.NewDocSet(cwd)
	if err != nil {
		log.Fatalf("Error building docset: %v\n", err)
	}
	fmt.Printf("Root: %s\n", d.Root)

	templates := loadTemplates(d)

	headerStr, headerMod := incMust(Include(d, "header.html", "head"))
	inlineCss, inlineCssMod := incMust(SimpleInclude(d, "css/b.css"))
	footerStr, footerMod := incMust(Include(d, "footer.html", "body"))
	titlebarStr, titlebarMod := incMust(Include(d, "titlebar.html", "body"))

	entryMod := modifiedTime(filepath.Join(d.Root, "tpl", "entry.html"))

	incMod := Newest(headerMod, inlineCssMod, footerMod, titlebarMod, entryMod)

	oneentry := make([]*Entry, 1)
	data := &TemplateData{
		Domain:    DOMAIN,
		SiteTitle: SITE_TITLE,
		Header:    headerStr,
		InlineCSS: string(inlineCss),
		Titlebar:  titlebarStr,
		Footer:    footerStr,
		Entries:   oneentry,
	}

	entries := make([]*Entry, 0)

	// Walk the docset and copy over files, possibly transformed.  Collect all
	// the entries along the way.
	walker := func(path string, info os.FileInfo, err error) error {
		attr, err := d.Path(path)
		if err != nil {
			return err
		}
		if info.IsDir() && attr.Has(piccolo.IGNORE) {
			return filepath.SkipDir
		}
		dest, err := d.Dest(path)
		if err != nil {
			return err
		}
		destMod := modifiedTime(dest)
		if !info.IsDir() && attr.Has(piccolo.INCLUDE) {
			if filepath.Ext(path) == ".html" {
				fileinfo, err := piccolo.CreationDateSaved(path)
				if err != nil {
					return err
				}
				if err := piccolo.LaTex(fileinfo.Node); err != nil {
					fmt.Printf("Error: expanding LaTex: %s", err)
				}
				url, err := d.URL(path)
				if err != nil {
					return err
				}
				entries = append(entries, &Entry{
					Path:    path,
					Title:   fileinfo.Title,
					URL:     url,
					Created: fileinfo.Created,
					Updated: fileinfo.Updated,
				})
				if Newest(fileinfo.Updated, incMod).After(destMod) {
					fmt.Printf("INCLUDE:  %v\n", dest)

					// Use the data for template expansion, but with only one entry in it.
					data.Entries[0] = entries[len(entries)-1]
					data.Entries[0].Body = StrFromNodes(fileinfo.Body())
					if err := Expand(d, templates.EntryHTML, data, path); err != nil {
						return err
					}
				}
			}
		}
		if !info.IsDir() && attr.Has(piccolo.VERBATIM) {
			if info.ModTime().After(destMod) {
				fmt.Printf("VERBATIM: %v\n", dest)
				if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
					return err
				}
				dst, err := os.Create(dest)
				if err != nil {
					return err
				}
				defer dst.Close()
				src, err := os.Open(path)
				if err != nil {
					return err
				}
				defer src.Close()
				_, err = io.Copy(dst, src)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	err = filepath.Walk(d.Root, walker)
	if err != nil {
		fatalf("Error walking: %v\n", err)
	}

	sort.Sort(EntryByCreated(entries))
	data.Entries = entries

	// TODO(jcgregorio) This is actually wrong, need to sort by Updated first, as if anyone cares.
	data.Updated = entries[0].Updated

	if err := Expand(d, templates.ArchiveHTML, data, filepath.Join(d.Archive, "index.html")); err != nil {
		fatalf("Error building archive: %v\n", err)
	}

	// Take the first 10 items from the list, expand the Body, then pass to templates.
	latest := entries[:FEED_LEN]
	for _, e := range latest {
		fi, _ := piccolo.CreationDateSaved(e.Path)
		if err := piccolo.LaTex(fi.Node); err != nil {
			fmt.Printf("Error: expanding LaTex: %s", err)
		}
		e.Body = StrFromNodes(fi.Body())
	}
	data.Entries = latest

	if err := Expand(d, templates.IndexHTML, data, filepath.Join(d.Main, "index.html")); err != nil {
		fatalf("Error building archive: %v\n", err)
	}

	if err := Expand(d, templates.IndexAtom, data, filepath.Join(d.Feed, "index.atom")); err != nil {
		fatalf("Error building feed: %v\n", err)
	}
}
