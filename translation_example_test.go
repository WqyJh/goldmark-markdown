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
	renderer := markdown.NewRenderer(markdown.WithTextTransformer(markdown.MapTransformer(translations)))

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
	renderer := markdown.NewRenderer(markdown.WithTextTransformer(markdown.MapTransformer(translations)))

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

// TestInput1Translation tests the translation of content from the input1.txt file
func TestInput1Translation(t *testing.T) {
	// Create a source markdown document from input1.txt
	source := []byte(`# Welcome to Our Documentation

This is a sample markdown file that contains various markdown elements.

## Features

- **Bold text** and *italic text*
- Code blocks with syntax highlighting
- Lists and sublists
- Links and images

### Code Example

` + "```python" + `
def hello_world():
    print("Hello, World!")
` + "```" + `

### Links and Images

- [Visit our website](https://example.com)
- ![Sample Image](https://example.com/image.jpg)

## Tables

| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |

> This is a blockquote
> With multiple lines

---

*Last updated: 2024*`)

	// Create a translation map
	translations := map[string]string{
		"Welcome to Our Documentation":                                            "欢迎访问我们的文档",
		"This is a sample markdown file that contains various markdown elements.": "这是一个包含各种 Markdown 元素的示例文件。",
		"Features":                             "功能特点",
		"Bold text":                            "粗体文本",
		"and":                                  "和",
		"italic text":                          "斜体文本",
		"Code blocks with syntax highlighting": "带有语法高亮的代码块",
		"Lists and sublists":                   "列表和子列表",
		"Links and images":                     "链接和图片",
		"Code Example":                         "代码示例",
		"Links and Images":                     "链接和图片",
		"Visit our website":                    "访问我们的网站",
		"Sample Image":                         "示例图片",
		"Tables":                               "表格",
		"Header 1":                             "标题 1",
		"Header 2":                             "标题 2",
		"Cell 1":                               "单元格 1",
		"Cell 2":                               "单元格 2",
		"Cell 3":                               "单元格 3",
		"Cell 4":                               "单元格 4",
		"This is a blockquote\nWith multiple lines": "这是一个引用\n包含多行",
		"Last updated: 2024":                        "最后更新：2024",
	}

	// Create a markdown renderer with translations
	renderer := markdown.NewRenderer(markdown.WithTextTransformer(markdown.MapTransformer(translations)))

	// Create a new goldmark instance with our renderer
	md := goldmark.New(
		goldmark.WithRenderer(renderer),
		goldmark.WithExtensions(renderer),
	)

	// Convert the markdown to a new document
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatalf("Failed to convert markdown: %v", err)
	}

	// Print the result for analysis
	result := buf.String()
	fmt.Println(result)

	// Expected result based on our translation rules
	expected := `# 欢迎访问我们的文档

这是一个包含各种 Markdown 元素的示例文件。

## 功能特点

- **粗体文本** 和 *斜体文本*
- 带有语法高亮的代码块
- 列表和子列表
- 链接和图片

### 代码示例

` + "```python" + `
def hello_world():
    print("Hello, World!")
` + "```" + `

### 链接和图片

- [访问我们的网站](https://example.com)
- ![示例图片](https://example.com/image.jpg)

## 表格
| 标题 1 | 标题 2 |
| ----- | ----- |
| 单元格 1 | 单元格 2 |
| 单元格 3 | 单元格 4 |

> 这是一个引用
> 包含多行

---

*最后更新：2024*
`

	if result != expected {
		t.Errorf("Expected:\n%q\nGot:\n%q", expected, result)
	}
}
