package piccolo

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"go.skia.org/infra/go/exec"

	"golang.org/x/net/html"
)

// LaTex finds <latex-pic> nodes in the html and
// replaces them with PNG images of the rendered LaTex.
func LaTex(node *html.Node) error {
	var f func(*html.Node) error
	f = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == "latex-pic" {
			fmt.Printf("%q", n.FirstChild.Data)
			file, err := ioutil.TempFile("/tmp", "piccolo-latex-")
			if err != nil {
				return fmt.Errorf("Couldn't create temp file: %s", err)
			}
			//defer os.Remove(file.Name())
			_, err = file.Write([]byte(n.FirstChild.Data))
			if err != nil {
				return fmt.Errorf("Failed to write file: %s", err)
			}
			file.Close()
			// Convert the latex to a PNG with:
			//
			//   tex2im  -z -a -o ./dst/test.png test.tex
			dest, err := ioutil.TempFile("/tmp", "piccolo-latex-")
			if err != nil {
				return fmt.Errorf("Couldn't create temp file: %s", err)
			}
			dest.Close()
			args := fmt.Sprintf("-z -a -o %s %s", dest.Name(), file.Name())
			output := bytes.Buffer{}
			err = exec.Run(&exec.Command{
				Name: "tex2im",
				Args: strings.Split(args, " "),
				// Set environment:
				Env: []string{},
				// Set working directory:
				CombinedOutput: &output,
				// Set a timeout:
				Timeout:     10 * time.Minute,
				InheritPath: true,
			})
			if err != nil {
				return fmt.Errorf("Failed to run tex2im: %q %s", output, err)
			}
			b, err := ioutil.ReadFile(dest.Name())
			if err != nil {
				return fmt.Errorf("Failed to read PNG: %s", err)
			}
			b64 := base64.StdEncoding.EncodeToString(b)
			uri := fmt.Sprintf("data:image/png;base64,%s", b64)
			fmt.Println(uri)

			//
			// Then embed the image via data: URI.

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
				},
			}
			n.Parent.InsertBefore(imgNode, n)
			n.Parent.RemoveChild(n)
			return nil
			// Delete the latex-pic element.
			// Add an img node.
			// Set src to the data uri.
			// Set alt to the LaTeX code.
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		return nil
	}
	return f(node)
}
