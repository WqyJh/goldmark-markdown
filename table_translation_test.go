package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

func TestTableTranslation(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		translations map[string]string
		expected     string
	}{
		{
			name: "simple table translation",
			source: "| Header 1 | Header 2 |\n" +
				"|---------|----------|\n" +
				"| Cell 1  | Cell 2   |",
			translations: map[string]string{
				"Header 1": "标题 1",
				"Header 2": "标题 2",
				"Cell 1":   "单元格 1",
				"Cell 2":   "单元格 2",
			},
			expected: "| 标题 1 | 标题 2 |\n" +
				"| ----- | ----- |\n" +
				"| 单元格 1 | 单元格 2 |\n",
		},
		{
			name: "aligned table translation",
			source: "| Left | Center | Right |\n" +
				"|:-----|:------:|------:|\n" +
				"| 1    | 2      | 3     |",
			translations: map[string]string{
				"Left":   "左对齐",
				"Center": "居中",
				"Right":  "右对齐",
				"1":      "一",
				"2":      "二",
				"3":      "三",
			},
			expected: "| 左对齐 | 居中 | 右对齐 |\n" +
				"| :----- | :----: | -----: |\n" +
				"| 一 | 二 | 三 |\n",
		},
		{
			name: "table with formatting",
			source: "| *Italic* | **Bold** |\n" +
				"|---------|----------|\n" +
				"| `Code`  | [Link](http://example.com) |",
			translations: map[string]string{
				"Italic": "斜体",
				"Bold":   "粗体",
				"Code":   "代码",
				"Link":   "链接",
			},
			expected: "| *斜体* | **粗体** |\n" +
				"| ----- | ----- |\n" +
				"| `Code` | [链接](http://example.com) |\n",
		},
		{
			name: "mixed translation table",
			source: "| Header 1 | Header 2 |\n" +
				"|---------|----------|\n" +
				"| Cell 1  | NotTranslated |",
			translations: map[string]string{
				"Header 1": "标题 1",
				"Header 2": "标题 2",
				"Cell 1":   "单元格 1",
				// NotTranslated not in translations map
			},
			expected: "| 标题 1 | 标题 2 |\n" +
				"| ----- | ----- |\n" +
				"| 单元格 1 | NotTranslated |\n",
		},
		{
			name: "multi-row table translation",
			source: "| Header 1 | Header 2 |\n" +
				"|---------|----------|\n" +
				"| Row 1 Cell 1 | Row 1 Cell 2 |\n" +
				"| Row 2 Cell 1 | Row 2 Cell 2 |",
			translations: map[string]string{
				"Header 1":     "标题 1",
				"Header 2":     "标题 2",
				"Row 1 Cell 1": "第1行单元格1",
				"Row 1 Cell 2": "第1行单元格2",
				"Row 2 Cell 1": "第2行单元格1",
				"Row 2 Cell 2": "第2行单元格2",
			},
			expected: "| 标题 1 | 标题 2 |\n" +
				"| ----- | ----- |\n" +
				"| 第1行单元格1 | 第1行单元格2 |\n" +
				"| 第2行单元格1 | 第2行单元格2 |\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new markdown renderer with translations
			rd := NewRenderer(WithTranslations(tt.translations))

			// Create Goldmark with our renderer and table extension
			md := goldmark.New(
				goldmark.WithRenderer(rd),
				goldmark.WithExtensions(rd),
			)

			var buf bytes.Buffer
			err := md.Convert([]byte(tt.source), &buf)
			if err != nil {
				t.Fatalf("Failed to convert markdown: %v", err)
			}

			// Check the result
			result := buf.String()
			require.Equal(t, tt.expected, result, "expected: %q, got: %q", tt.expected, result)
		})
	}
}
