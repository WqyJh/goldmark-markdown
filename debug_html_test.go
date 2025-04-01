package markdown

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// DebugVisitor visits each node in the AST and prints HTML node details
type DebugVisitor struct {
	source []byte
	w      *bytes.Buffer
}

func (v *DebugVisitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	if !entering {
		return ast.WalkContinue
	}

	// Print node type
	fmt.Fprintf(v.w, "Node: %T\n", node)

	if raw, ok := node.(*ast.RawHTML); ok {
		fmt.Fprintf(v.w, "RawHTML with %d segments:\n", raw.Segments.Len())
		for i := 0; i < raw.Segments.Len(); i++ {
			segment := raw.Segments.At(i)
			content := segment.Value(v.source)
			fmt.Fprintf(v.w, "  Segment %d: %q\n", i, content)
		}
	}

	if htmlBlock, ok := node.(*ast.HTMLBlock); ok {
		fmt.Fprintf(v.w, "HTMLBlock with %d lines:\n", htmlBlock.Lines().Len())
		for i := 0; i < htmlBlock.Lines().Len(); i++ {
			line := htmlBlock.Lines().At(i)
			content := line.Value(v.source)
			fmt.Fprintf(v.w, "  Line %d: %q\n", i, content)
		}
		if htmlBlock.HasClosure() {
			fmt.Fprintf(v.w, "  Closure: %q\n", htmlBlock.ClosureLine.Value(v.source))
		}
	}

	return ast.WalkContinue
}

func TestDebugHTMLNodes(t *testing.T) {
	sources := []struct {
		name   string
		source string
	}{
		{
			name:   "inline html",
			source: "Before <div class=\"test\">HTML content</div> after",
		},
		{
			name:   "html block",
			source: "Before\n\n<div>\n  <p>Block HTML</p>\n</div>\n\nAfter",
		},
	}

	for _, src := range sources {
		t.Run(src.name, func(t *testing.T) {
			// Parse the source
			md := goldmark.New()
			reader := text.NewReader([]byte(src.source))
			doc := md.Parser().Parse(reader)

			// Visit the AST
			var buf bytes.Buffer
			visitor := &DebugVisitor{
				source: []byte(src.source),
				w:      &buf,
			}
			err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				return visitor.Visit(n, entering), nil
			})
			if err != nil {
				t.Fatal(err)
			}

			// Print the results
			fmt.Println("Debug for:", src.name)
			fmt.Println(buf.String())
			t.Logf("Debug output for %s:\n%s", src.name, buf.String())
		})
	}
}
