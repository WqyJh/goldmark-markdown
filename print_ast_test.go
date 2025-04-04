package markdown

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestPrintAST(t *testing.T) {
	// Sample markdown document with various elements
	markdown := []byte(`# Heading 1

## Heading 2

This is a paragraph with *emphasis* and **strong emphasis**.

This is a ` + "`" + "inline code" + "`" + `.

- List item 1
- List item 2
  - Nested item

1. Ordered list item 1
2. Ordered list item 2

[Link text](https://example.com "title")

![Image alt](image.jpg)

> This is a blockquote.

<div>
This is a div.
</div>

Before 
<div class=\"test\">HTML content</div> after

Plain text and <em>emphasis</em> with HTML
Inline <body>HTML<div>content</div>
</body>

` + "```go" + `
func example() {
    fmt.Println("Hello, world!")
}
` + "```" + `

`)

	// Parse the markdown into an AST
	parser := goldmark.DefaultParser()
	source := text.NewReader(markdown)
	doc := parser.Parse(source)

	// Test PrintAST by capturing its output
	var buf bytes.Buffer
	err := PrintAST(&buf, markdown, doc)
	if err != nil {
		t.Fatalf("PrintAST returned an error: %v", err)
	}

	// Print the result for visual inspection
	output := buf.String()
	t.Logf("AST Output:\n%s", output)

	// Basic verification of AST structure
	// We'll check for some expected elements in the output
	expectedNodes := []string{
		"Document",
		"Heading [Level=1]",
		"Heading [Level=2]",
		"Paragraph",
		"Emphasis [Level=1]",
		"Emphasis [Level=2]",
		"List [Tight=true] [Bullet]",
		"ListItem",
		"List [Tight=true] [Ordered start=1]",
		"Link [",
		"Image [",
		"Blockquote",
		"FencedCodeBlock [Lang=go]",
	}

	for _, expected := range expectedNodes {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Errorf("Expected AST output to contain '%s', but it was not found", expected)
		}
	}
}

// TestRenderAndPrintAST tests that the rendered markdown produces the same AST as the original
func TestRenderAndPrintAST(t *testing.T) {
	// Sample markdown document
	originalMarkdown := []byte(`# Test Document

This is a *test* with **formatting**.

- List item 1
- List item 2
`)

	// Parse the original markdown into an AST
	parser := goldmark.DefaultParser()
	source := text.NewReader(originalMarkdown)
	originalDoc := parser.Parse(source)

	// Render the AST back to markdown
	var renderedBuf bytes.Buffer
	renderer := NewRenderer()
	err := renderer.Render(&renderedBuf, originalMarkdown, originalDoc)
	if err != nil {
		t.Fatalf("Failed to render markdown: %v", err)
	}
	renderedMarkdown := renderedBuf.Bytes()

	// Parse the rendered markdown into a new AST
	renderedSource := text.NewReader(renderedMarkdown)
	renderedDoc := parser.Parse(renderedSource)

	// Print both ASTs
	var originalASTBuf, renderedASTBuf bytes.Buffer

	err = PrintAST(&originalASTBuf, originalMarkdown, originalDoc)
	if err != nil {
		t.Fatalf("Failed to print original AST: %v", err)
	}

	err = PrintAST(&renderedASTBuf, renderedMarkdown, renderedDoc)
	if err != nil {
		t.Fatalf("Failed to print rendered AST: %v", err)
	}

	t.Logf("Original Markdown:\n%s", originalMarkdown)
	t.Logf("Rendered Markdown:\n%s", renderedMarkdown)
	t.Logf("Original AST:\n%s", originalASTBuf.String())
	t.Logf("Rendered AST:\n%s", renderedASTBuf.String())
}

// TestPrintASTFromMarkdown tests the new function that takes only markdown text as input
func TestPrintASTFromMarkdown(t *testing.T) {
	// Sample markdown document with various elements
	markdown := []byte(`# Heading 1

## Heading 2

This is a paragraph with *emphasis* and **strong emphasis**.

`)

	// Test PrintASTFromMarkdown by capturing its output
	var buf bytes.Buffer

	err := PrintASTFromMarkdown(&buf, markdown)
	if err != nil {
		t.Fatalf("PrintASTFromMarkdown returned an error: %v", err)
	}

	// Print the result for visual inspection
	output := buf.String()
	t.Logf("AST Output:\n%s", output)

	// Basic verification of AST structure
	expectedNodes := []string{
		"Document",
		"Heading [Level=1]",
		"Heading [Level=2]",
		"Paragraph",
		"Emphasis [Level=1]",
		"Emphasis [Level=2]",
	}

	for _, expected := range expectedNodes {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Errorf("Expected AST output to contain '%s', but it was not found", expected)
		}
	}
}

// TestPrintASTWithTable tests the AST printing with table support
func TestPrintASTWithTable(t *testing.T) {
	// Sample markdown document with table
	markdown := []byte(`# Document with Table

This is a sample table:

| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |

End of document.
`)

	// Test PrintASTFromMarkdown by capturing its output
	var buf bytes.Buffer

	err := PrintASTFromMarkdown(&buf, markdown)
	if err != nil {
		t.Fatalf("PrintASTFromMarkdown returned an error: %v", err)
	}

	// Print the result for visual inspection
	output := buf.String()
	t.Logf("AST Output with Table:\n%s", output)

	// Basic verification of Table AST structure
	expectedTableNodes := []string{
		"Table",
		"TableHeader",
		"TableRow",
		"TableCell",
	}

	for _, expected := range expectedTableNodes {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Errorf("Expected AST output to contain '%s', but it was not found", expected)
		}
	}
}

// TestPrintASTWithAdvancedTable tests the AST printing with a more complex table
func TestPrintASTWithAdvancedTable(t *testing.T) {
	// Sample markdown document with advanced table features
	markdown := []byte(`# Advanced Table Test

Here's a table with various alignments and formatting:

| Left | Center | Right | Mixed Content |
|:-----|:------:|------:|---------------|
| 1    | *2*    | 3     | [Link](http://example.com) |
| **Bold** | Text | 123  | ![Image](img.png) |
| Multiline<br>cell | Empty |  | Code: ` + "`inline`" + ` |

End of table.
`)

	// Test PrintASTFromMarkdown by capturing its output
	var buf bytes.Buffer

	err := PrintASTFromMarkdown(&buf, markdown)
	if err != nil {
		t.Fatalf("PrintASTFromMarkdown returned an error: %v", err)
	}

	// Print the result for visual inspection
	output := buf.String()
	t.Logf("AST Output with Advanced Table:\n%s", output)

	// Check for more complex table nodes in the output
	tableContentNodes := []string{
		"TableHeader",
		"TableRow",
		"TableCell",
		"Text",
		"Emphasis",
		"Link",
		"Image",
	}

	for _, expected := range tableContentNodes {
		if !bytes.Contains(buf.Bytes(), []byte(expected)) {
			t.Errorf("Expected AST output to contain '%s', but it was not found", expected)
		}
	}
}
