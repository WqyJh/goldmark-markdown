package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// PrintAST prints the AST structure of a Markdown document to the specified writer
func PrintAST(w io.Writer, source []byte, n ast.Node) error {
	_, err := fmt.Fprintln(w, "AST Tree:")
	if err != nil {
		return err
	}
	return printASTNode(w, source, n, 0, "")
}

// PrintASTFromMarkdown parses the markdown text into an AST and prints its structure
func PrintASTFromMarkdown(w io.Writer, source []byte) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
		),
	)
	parser := md.Parser()
	reader := text.NewReader(source)
	doc := parser.Parse(reader)

	return PrintAST(w, source, doc)
}

// printASTNode prints a single AST node and its children recursively with visual tree structure
func printASTNode(w io.Writer, source []byte, n ast.Node, level int, prefix string) error {
	// Create the appropriate prefix for this level
	var currentPrefix string
	if level > 0 {
		if prefix == "" {
			currentPrefix = "└── "
		} else {
			currentPrefix = prefix + "└── "
		}
	}

	// Print node type
	nodeName := fmt.Sprintf("%T", n)
	// Extract just the type name without package
	if idx := strings.LastIndex(nodeName, "."); idx >= 0 {
		nodeName = nodeName[idx+1:]
	}

	fmt.Fprintf(w, "%s%s", prefix+currentPrefix, nodeName)

	// Print additional attributes based on node type
	switch n := n.(type) {
	case *ast.Text:
		fmt.Fprintf(w, " [%q]", n.Value(source))
	case *ast.String:
		fmt.Fprintf(w, " [%q]", n.Value)
	case *ast.RawHTML:
		fmt.Fprintf(w, " [HTML]")
	case *ast.Link:
		fmt.Fprintf(w, " [%s]", n.Destination)
	case *ast.Image:
		fmt.Fprintf(w, " [%s]", n.Destination)
	case *ast.Heading:
		fmt.Fprintf(w, " [Level=%d]", n.Level)
	case *ast.ListItem:
		fmt.Fprintf(w, " [%d]", n.Offset)
	case *ast.List:
		fmt.Fprintf(w, " [Tight=%t]", n.IsTight)
		if n.IsOrdered() {
			fmt.Fprintf(w, " [Ordered start=%d]", n.Start)
		} else {
			fmt.Fprintf(w, " [Bullet]")
		}
	case *ast.CodeSpan:
		fmt.Fprintf(w, " [Code]")
	case *ast.Emphasis:
		fmt.Fprintf(w, " [Level=%d]", n.Level)
	case *ast.FencedCodeBlock:
		if n.Info != nil {
			fmt.Fprintf(w, " [Lang=%s]", n.Info.Value(source))
		}
		// Print code content
		if n.Lines().Len() > 0 {
			fmt.Fprintf(w, " Content:")
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				fmt.Fprintf(w, "\n%s%s  |%s", prefix+currentPrefix, strings.Repeat(" ", len(nodeName)), line.Value(source))
			}
		}
	case *ast.CodeBlock:
		// Print code content
		if n.Lines().Len() > 0 {
			fmt.Fprintf(w, " Content:")
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				fmt.Fprintf(w, "\n%s%s  |%s", prefix+currentPrefix, strings.Repeat(" ", len(nodeName)), line.Value(source))
			}
		}
	case *east.Table:
		fmt.Fprintf(w, " [Table]")
	case *east.TableHeader:
		fmt.Fprintf(w, " [Header Row]")
	case *east.TableRow:
		fmt.Fprintf(w, " [Row]")
	case *east.TableCell:
		fmt.Fprintf(w, " [Cell]")
	}

	fmt.Fprintln(w)

	// Print children recursively
	lastChild := n.LastChild()
	childPrefix := prefix
	if level > 0 {
		if prefix == "" {
			childPrefix = "    "
		} else {
			childPrefix = prefix + "    "
		}
	}

	for c := n.FirstChild(); c != nil; {
		nextChild := c.NextSibling()

		// Use different prefixes for the last child
		var newPrefix string
		if c == lastChild {
			newPrefix = childPrefix
		} else {
			newPrefix = childPrefix
			if level > 0 {
				newPrefix = strings.TrimRight(childPrefix, " ") + "│   "
			}
		}

		if err := printASTNode(w, source, c, level+1, newPrefix); err != nil {
			return err
		}

		c = nextChild
	}

	return nil
}
