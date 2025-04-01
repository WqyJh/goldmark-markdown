package markdown

import (
	"bytes"
	"os"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestTranslations(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		translations map[string]string
		expected     string
	}{
		{
			name:         "simple text translation",
			source:       "Hello world",
			translations: map[string]string{"Hello world": "你好世界"},
			expected:     "你好世界\n",
		},
		{
			name:         "paragraph translation",
			source:       "This is a test.\n\nAnother paragraph.",
			translations: map[string]string{"This is a test.": "这是一个测试。", "Another paragraph.": "另一个段落。"},
			expected:     "这是一个测试。\n\n另一个段落。\n",
		},
		{
			name:         "heading translation",
			source:       "# Title\n\nContent",
			translations: map[string]string{"Title": "标题", "Content": "内容"},
			expected:     "# 标题\n\n内容\n",
		},
		{
			name:         "image alt text translation",
			source:       "![Image Title](image.jpg)",
			translations: map[string]string{"Image Title": "图片标题"},
			expected:     "![图片标题](image.jpg)\n",
		},
		{
			name: "complex markdown translation",
			source: "# Document Title\n\n" +
				"## Introduction\n\n" +
				"This is a complex document with *italic* and **bold** text.\n\n" +
				"- List item 1\n" +
				"- List item 2\n" +
				"  - Nested item 2.1\n" +
				"  - Nested item 2.2\n" +
				"- List item 3\n\n" +
				"1. Ordered item 1\n" +
				"2. Ordered item 2\n\n" +
				"> This is a blockquote\n" +
				"> With multiple lines\n\n" +
				"Here's a [link](https://example.com) and some `inline code`.\n\n" +
				"    // Code block\n" +
				"    function hello() {\n" +
				"      return \"world\";\n" +
				"    }\n\n" +
				"### Conclusion\n\n" +
				"Final paragraph with some important information.",
			translations: map[string]string{
				"Document Title":                  "文档标题",
				"Introduction":                    "介绍",
				"This is a complex document with": "这是一个复杂的文档，包含",
				"italic":                          "斜体",
				"and":                             "和",
				"bold":                            "粗体",
				"text.":                           "文本。",
				"List item 1":                     "列表项 1",
				"List item 2":                     "列表项 2",
				"Nested item 2.1":                 "嵌套项 2.1",
				"Nested item 2.2":                 "嵌套项 2.2",
				"List item 3":                     "列表项 3",
				"Ordered item 1":                  "有序项 1",
				"Ordered item 2":                  "有序项 2",
				"This is a blockquote":            "这是一个引用",
				"With multiple lines":             "包含多行",
				"Here's a":                        "这是一个",
				"link":                            "链接",
				"and some":                        "和一些",
				"inline code":                     "内联代码",
				"Conclusion":                      "结论",
				"Final paragraph with some important information.": "最后的段落包含一些重要信息。",
			},
			expected: "# 文档标题\n\n" +
				"## 介绍\n\n" +
				"这是一个复杂的文档，包含 *斜体* 和 **粗体** 文本。\n\n" +
				"- 列表项 1\n" +
				"- 列表项 2\n" +
				"  - 嵌套项 2.1\n" +
				"  - 嵌套项 2.2\n" +
				"- 列表项 3\n\n" +
				"1. 有序项 1\n" +
				"2. 有序项 2\n\n" +
				"> 这是一个引用\n" +
				"> 包含多行\n\n" +
				"这是一个 [链接](https://example.com) 和一些 `inline code`.\n\n" +
				"    // Code block\n" +
				"    function hello() {\n" +
				"      return \"world\";\n" +
				"    }\n\n" +
				"### 结论\n\n" +
				"最后的段落包含一些重要信息。\n",
		},
		{
			name:         "no translations",
			source:       "Hello world",
			translations: map[string]string{},
			expected:     "Hello world\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintASTFromMarkdown(os.Stdout, []byte(tt.source))
			// Create a new markdown renderer with translations
			renderer := NewRenderer(WithTranslations(tt.translations))

			// Parse the markdown source
			doc := goldmark.New(
				goldmark.WithRenderer(renderer),
			)

			var buf bytes.Buffer
			err := doc.Convert([]byte(tt.source), &buf)
			if err != nil {
				t.Fatalf("Failed to convert markdown: %v", err)
			}

			// Check the result
			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

// This test directly tests the renderText method to ensure translations work at that level
func TestRenderText(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		translations    map[string]string
		skipTranslation bool
		expected        string
	}{
		{
			name:            "simple translation",
			text:            "Hello",
			translations:    map[string]string{"Hello": "你好"},
			skipTranslation: false,
			expected:        "你好",
		},
		{
			name:            "no matching translation",
			text:            "Hello",
			translations:    map[string]string{"World": "世界"},
			skipTranslation: false,
			expected:        "Hello",
		},
		{
			name:            "empty translation map",
			text:            "Hello",
			translations:    map[string]string{},
			skipTranslation: false,
			expected:        "Hello",
		},
		{
			name:            "skip translation",
			text:            "Hello",
			translations:    map[string]string{"Hello": "你好"},
			skipTranslation: true,
			expected:        "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := NewConfig(WithTranslations(tt.translations))
			renderer := NewRenderer(WithTranslations(tt.translations))
			source := []byte(tt.text)

			// Create a text node
			textNode := ast.NewText()
			segment := text.NewSegment(0, len(source))
			textNode.Segment = segment

			// Render the text node
			ctx := newRenderContext(&buf, source, config)
			ctx.skipTranslation = tt.skipTranslation
			renderer.rc = ctx
			renderer.renderText(textNode, true)
			// Flush the writer to ensure all content is written to the buffer
			ctx.writer.FlushLine()

			result := buf.String()
			result = result[:len(result)-1] // Remove trailing newline added by FlushLine
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}
