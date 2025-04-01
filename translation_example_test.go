package markdown_test

import (
	"bytes"
	"fmt"
	"testing"

	markdown "github.com/teekennedy/goldmark-markdown"
	"github.com/yuin/goldmark"
)

// TestTranslationSkipping demonstrates how the translation skipping works for specific node types
func TestTranslationSkipping(t *testing.T) {
	// Create a source markdown document with different elements
	source := []byte("# Test Title\n\nThis is a paragraph with *italic* and **bold** text.\n\nA [link](https://example.com) and some `inline code`.\n\n> A blockquote\n\n```\ncode block\n```\n\n<div>HTML block</div>\n")

	// Create a translation map
	translations := map[string]string{
		"Test Title":   "测试标题",
		"italic":       "斜体",
		"bold":         "粗体",
		"link":         "链接",
		"A blockquote": "一个引用",
	}

	// Create a markdown renderer with translations
	renderer := markdown.NewRenderer(markdown.WithTranslations(translations))

	// Create a new goldmark instance with our renderer
	md := goldmark.New(
		goldmark.WithRenderer(renderer),
	)

	// Convert the markdown to a new document
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	// Print the result for analysis
	result := buf.String()
	fmt.Println(result)

	// Check that only appropriate parts are translated
	expected := "# 测试标题\n\nThis is a paragraph with *斜体* and **粗体** text.\n\nA [链接](https://example.com) and some `inline code`.\n\n> 一个引用\n\n```\ncode block\n```\n\n<div>HTML block</div>\n"
	if result != expected {
		t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
	}
}

// TestSpecialTranslationRules verifies that the specific translation rules are applied correctly
func TestSpecialTranslationRules(t *testing.T) {
	// Create a source markdown document with the elements we want to test
	source := []byte("# Test Header\n\nThis is normal text with a [link text](https://example.com) and <https://auto.link>.\n\nThis is an ![image alt text](https://example.com/image.jpg) with title.\n\nThis has `code span` that shouldn't be translated.\n\n```go\n// Code block that shouldn't be translated\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n<div>Raw HTML should not be translated</div>\n")

	// Create a translation map - adjust to match actual text nodes
	translations := map[string]string{
		"Test Header":                       "测试标题",
		"This is normal text with a":        "这是普通文本带有",
		"link text":                         "链接文本",
		"and":                               "和",
		"This is an":                        "这是一个",
		"image alt text":                    "图片替代文本",
		"Image Title":                       "图片标题",
		"with title.":                       "带标题.",
		"This has":                          "这有",
		"code span":                         "代码段", // This should NOT be translated
		"that shouldn't be translated.":     "不应该被翻译.",
		"Raw HTML should not be translated": "原始HTML不应该被翻译", // This should NOT be translated
	}

	// Create a markdown renderer with translations
	renderer := markdown.NewRenderer(markdown.WithTranslations(translations))

	// Create a new goldmark instance with our renderer
	md := goldmark.New(
		goldmark.WithRenderer(renderer),
	)

	// Convert the markdown to a new document
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	// Print the result for analysis
	result := buf.String()
	fmt.Println(result)

	// Expected result based on our rules and actual behavior:
	// - Link and image text should be translated, but URLs should not
	// - CodeSpan, FencedCodeBlock, and HTML should not be translated
	expected := "# 测试标题\n\n这是普通文本带有 [链接文本](https://example.com) 和 <https://auto.link>.\n\n这是一个 ![图片替代文本](https://example.com/image.jpg) 带标题.\n\n这有 `code span` 不应该被翻译.\n\n```go\n// Code block that shouldn't be translated\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n<div>Raw HTML should not be translated</div>\n"

	if result != expected {
		t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
	}
}
