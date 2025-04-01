package markdown

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
)

// HTMLTransformer is a simple implementation of TextTransformer that keeps track of transform calls
type HTMLTransformer struct {
	Translations           map[string]string
	HTMLTransformed        bool
	HTMLTransformedContent string
}

func (t *HTMLTransformer) Transform(textType TextType, text string) (string, bool) {
	if textType == TextTypeHTML {
		t.HTMLTransformed = true
		t.HTMLTransformedContent = text
		if translated, ok := t.Translations[text]; ok {
			return translated, true
		}
	} else if textType == TextTypePlain {
		if translated, ok := t.Translations[text]; ok {
			return translated, true
		}
	}
	return text, false
}

func TestHTMLTransformations(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		translations map[string]string
		expected     string
		htmlContent  string // Expected HTML content passed to transformer
	}{
		{
			name:   "raw html transformation",
			source: "Before \n<div class=\"test\">HTML content</div> after",
			translations: map[string]string{
				"<div class=\"test\">HTML content</div>": "<div class=\"test\">HTML content</div>",
				"Before":                                 "之前",
				"after":                                  "之后",
			},
			expected:    "之前\n<div class=\"test\">HTML content</div> after\n",
			htmlContent: "<div class=\"test\">HTML content</div> after",
		},
		{
			name:   "html block transformation",
			source: "Before\n\n<div>\n  <p>Block HTML</p>\n</div>\n\nAfter",
			translations: map[string]string{
				"<div>\n  <p>Block HTML</p>\n</div>\n": "<div>\n  <p>块 HTML</p>\n</div>\n",
				"Before":                               "之前",
				"After":                                "之后",
			},
			expected:    "之前\n\n<div>\n  <p>块 HTML</p>\n</div>\n\n之后\n",
			htmlContent: "<div>\n  <p>Block HTML</p>\n</div>\n",
		},
		{
			name:   "mixed html and text transformation",
			source: "Plain text and <em>emphasis</em> with HTML",
			translations: map[string]string{
				"Plain text and": "普通文本和",
				"emphasis":       "强调",
				"with HTML":      "带有HTML",
				"<em>":           "<<em>>",
				"</em>":          "<</em>>",
			},
			expected:    "普通文本和 <<em>>强调<</em>> 带有HTML\n",
			htmlContent: "</em>",
		},
		{
			name:   "no html transformation",
			source: "Plain text only",
			translations: map[string]string{
				"Plain text only": "仅纯文本",
			},
			expected: "仅纯文本\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTML transformer with the test translations
			transformer := &HTMLTransformer{
				Translations: tt.translations,
			}

			// Debug: Print the AST to see how HTML is being parsed
			t.Logf("AST for %s:", tt.name)
			PrintASTFromMarkdown(bytes.NewBufferString(""), []byte(tt.source))

			// Create a new markdown renderer with the transformer
			renderer := NewRenderer(WithTextTransformer(transformer))

			// Parse and render the markdown source
			md := goldmark.New(
				goldmark.WithRenderer(renderer),
			)

			var buf bytes.Buffer
			err := md.Convert([]byte(tt.source), &buf)
			if err != nil {
				t.Fatalf("Failed to convert markdown: %v", err)
			}

			// Check the result
			result := buf.String()
			require.Equal(t, tt.expected, result, "expected output:\n%q\nGot:\n%q", tt.expected, result)

			// For HTML tests, verify the HTML content was passed to the transformer
			if tt.htmlContent != "" {
				require.True(t, transformer.HTMLTransformed, "HTML content was not passed to TextTransformer")
				require.Equal(t, tt.htmlContent, transformer.HTMLTransformedContent, "expected HTML content passed to transformer:\n%q\nGot:\n%q", tt.htmlContent, transformer.HTMLTransformedContent)
			}
		})
	}
}
