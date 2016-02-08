package piccolo

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"go.skia.org/infra/go/exec"
	"golang.org/x/net/html"
)

// LaTex finds <latex-pic> nodes in the html and
// replaces them with PNG images of the rendered LaTex.
func LaTex(node *html.Node) error {
	latexNodes := []*html.Node{}
	var f func(*html.Node) error
	f = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == "latex-pic" {
			// Create a tmp file to write the Latex code into.
			file, err := ioutil.TempFile("/tmp", "piccolo-latex-")
			if err != nil {
				return fmt.Errorf("Couldn't create temp file: %s", err)
			}
			_, err = file.Write([]byte(n.FirstChild.Data))
			if err != nil {
				return fmt.Errorf("Failed to write file: %s", err)
			}
			file.Close()
			defer os.Remove(file.Name())
			// And create a tmp file to receive the PNG.
			dest, err := ioutil.TempFile("/tmp", "piccolo-latex-")
			if err != nil {
				return fmt.Errorf("Couldn't create temp file: %s", err)
			}
			dest.Close()
			defer os.Remove(dest.Name())
			// Convert the latex to a PNG with:
			//
			//   tex2im  -z -a -o ./dst/test.png test.tex
			args := fmt.Sprintf("-z -a -o %s %s", dest.Name(), file.Name())
			output := bytes.Buffer{}
			err = exec.Run(&exec.Command{
				Name:           "tex2im",
				Args:           strings.Split(args, " "),
				Env:            []string{},
				CombinedOutput: &output,
				Timeout:        10 * time.Minute,
				InheritPath:    true,
			})
			if err != nil {
				return fmt.Errorf("Failed to run tex2im: %q %s", output, err)
			}
			b, err := ioutil.ReadFile(dest.Name())
			if err != nil {
				return fmt.Errorf("Failed to read PNG: %s", err)
			}
			uri := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(b))

			// Create an img node.
			imgNode := &html.Node{
				Type: html.ElementNode,
				Data: "img",
				Attr: []html.Attribute{
					html.Attribute{
						Key: "src",
						Val: uri,
					},
					html.Attribute{
						Key: "alt",
						Val: n.FirstChild.Data,
					},
					html.Attribute{
						Key: "title",
						Val: n.FirstChild.Data,
					},
				},
			}
			// Insert it just before the latex-pic element.
			n.Parent.InsertBefore(imgNode, n)
			// Remove the original latex-pic element later.
			latexNodes = append(latexNodes, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		return nil
	}
	err := f(node)
	for _, n := range latexNodes {
		n.Parent.RemoveChild(n)
	}
	return err

}
