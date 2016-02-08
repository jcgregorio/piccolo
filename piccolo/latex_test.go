package piccolo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/net/html"

	"github.com/stretchr/testify/assert"
)

func TestLaTex(t *testing.T) {

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v\n", err)
	}
	testCases := []struct {
		Filename string
	}{
		{"latex1.html"},
	}
	for _, tc := range testCases {
		path := filepath.Join(cwd, "tests", "src", tc.Filename)
		fi, _, _ := CreationDate(path)
		err := LaTex(fi.Node)
		assert.NoError(t, err)
		buf := bytes.NewBuffer([]byte{})
		html.Render(buf, fi.Node)
		fmt.Printf("converted:", buf.String())
		assert.Contains(t, buf.String(), "<img src=\"data:image/png;base64,")
	}
}
