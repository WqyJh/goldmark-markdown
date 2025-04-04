// Package markdown is a goldmark renderer that outputs markdown.
package markdown

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// NewRenderer returns a new markdown Renderer that is configured by default values.
func NewRenderer(options ...Option) *Renderer {
	r := &Renderer{
		config:               NewConfig(),
		rc:                   renderContext{},
		maxKind:              20, // a random number slightly larger than the number of default ast kinds
		nodeRendererFuncsTmp: map[ast.NodeKind]renderer.NodeRendererFunc{},
	}
	for _, opt := range options {
		opt.SetMarkdownOption(r.config)
	}
	return r
}

// Renderer is an implementation of renderer.Renderer that renders nodes as Markdown
type Renderer struct {
	config               *Config
	rc                   renderContext
	nodeRendererFuncsTmp map[ast.NodeKind]renderer.NodeRendererFunc
	maxKind              int
	nodeRendererFuncs    []nodeRenderer
	initSync             sync.Once
}

var _ renderer.Renderer = &Renderer{}

// AddOptions implements renderer.Renderer.AddOptions
func (r *Renderer) AddOptions(opts ...renderer.Option) {
	config := renderer.NewConfig()
	for _, opt := range opts {
		opt.SetConfig(config)
	}
	for name, value := range config.Options {
		r.config.SetOption(name, value)
	}

	// handle any config.NodeRenderers set by opts
	config.NodeRenderers.Sort()
	l := len(config.NodeRenderers)
	for i := l - 1; i >= 0; i-- {
		v := config.NodeRenderers[i]
		nr, _ := v.Value.(renderer.NodeRenderer)
		nr.RegisterFuncs(r)
	}
}

func (r *Renderer) Register(kind ast.NodeKind, fun renderer.NodeRendererFunc) {
	r.nodeRendererFuncsTmp[kind] = fun
	if int(kind) > r.maxKind {
		r.maxKind = int(kind)
	}
}

// Render implements renderer.Renderer.Render
func (r *Renderer) Render(w io.Writer, source []byte, n ast.Node) error {
	r.rc = newRenderContext(w, source, r.config)
	r.initSync.Do(func() {
		r.nodeRendererFuncs = make([]nodeRenderer, r.maxKind+1)
		// add default functions
		// blocks
		r.nodeRendererFuncs[ast.KindDocument] = r.renderBlockSeparator
		r.nodeRendererFuncs[ast.KindHeading] = r.chainRenderers(r.renderBlockSeparator, r.renderHeading)
		r.nodeRendererFuncs[ast.KindBlockquote] = r.chainRenderers(r.renderBlockSeparator, r.renderBlockquote)
		r.nodeRendererFuncs[ast.KindCodeBlock] = r.chainRenderers(r.renderBlockSeparator, r.renderCodeBlock)
		r.nodeRendererFuncs[ast.KindFencedCodeBlock] = r.chainRenderers(r.renderBlockSeparator, r.renderFencedCodeBlock)
		r.nodeRendererFuncs[ast.KindHTMLBlock] = r.chainRenderers(r.renderBlockSeparator, r.renderHTMLBlock)
		r.nodeRendererFuncs[ast.KindList] = r.chainRenderers(r.renderBlockSeparator, r.renderList)
		r.nodeRendererFuncs[ast.KindListItem] = r.chainRenderers(r.renderBlockSeparator, r.renderListItem)
		r.nodeRendererFuncs[ast.KindParagraph] = r.renderBlockSeparator
		r.nodeRendererFuncs[ast.KindTextBlock] = r.renderBlockSeparator
		r.nodeRendererFuncs[ast.KindThematicBreak] = r.chainRenderers(r.renderBlockSeparator, r.renderThematicBreak)

		// inlines
		r.nodeRendererFuncs[ast.KindAutoLink] = r.renderAutoLink
		r.nodeRendererFuncs[ast.KindCodeSpan] = r.renderCodeSpan
		r.nodeRendererFuncs[ast.KindEmphasis] = r.renderEmphasis
		r.nodeRendererFuncs[ast.KindImage] = r.renderImage
		r.nodeRendererFuncs[ast.KindLink] = r.renderLink
		r.nodeRendererFuncs[ast.KindRawHTML] = r.renderRawHTML
		r.nodeRendererFuncs[ast.KindText] = r.renderText
		// TODO: add KindString
		// r.nodeRendererFuncs[ast.KindString] = r.renderString

		for kind, fun := range r.nodeRendererFuncsTmp {
			r.nodeRendererFuncs[kind] = r.transform(fun)
		}
		r.nodeRendererFuncsTmp = nil
	})
	return ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		return r.nodeRendererFuncs[n.Kind()](n, entering), r.rc.writer.Err()
	})
}

func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.renderTableHeader)
	reg.Register(east.KindTableRow, r.renderTableRow)
	reg.Register(east.KindTableCell, r.renderTableCell)
}

// transform wraps a renderer.NodeRendererFunc to match the nodeRenderer function signature
func (r *Renderer) transform(fn renderer.NodeRendererFunc) nodeRenderer {
	return func(n ast.Node, entering bool) ast.WalkStatus {
		status, _ := fn(r.rc.writer, r.rc.source, n, entering)
		return status
	}
}

// nodeRenderer is a markdown node renderer func.
type nodeRenderer func(ast.Node, bool) ast.WalkStatus

func (r *Renderer) chainRenderers(renderers ...nodeRenderer) nodeRenderer {
	return func(node ast.Node, entering bool) ast.WalkStatus {
		var walkStatus ast.WalkStatus
		for i := range renderers {
			// go through renderers in reverse when exiting
			if !entering {
				i = len(renderers) - 1 - i
			}
			walkStatus = renderers[i](node, entering)
		}
		return walkStatus
	}
}

func (r *Renderer) renderBlockSeparator(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		// Add blank previous line if applicable
		if node.PreviousSibling() != nil && node.HasBlankPreviousLines() {
			r.rc.writer.EndLine()
		}
	} else {
		// Flush line buffer to complete line written by previous block
		r.rc.writer.FlushLine()
	}
	return ast.WalkContinue
}

func (r *Renderer) renderAutoLink(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.AutoLink)
	if entering {
		r.rc.writer.WriteBytes([]byte("<"))
		// Set skipTranslation to true only for the URL part
		r.rc.skipTranslation = true
		r.rc.writer.WriteBytes(n.URL(r.rc.source))
	} else {
		r.rc.writer.WriteBytes([]byte(">"))
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderBlockquote(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.rc.writer.PushPrefix([]byte("> "))
	} else {
		r.rc.writer.PopPrefix()
	}
	return ast.WalkContinue
}

func (r *Renderer) renderHeading(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.Heading)
	// Empty headings or headings above level 2 can only be ATX
	if !n.HasChildren() || n.Level > 2 {
		return r.renderATXHeading(n, entering)
	}
	// Multiline headings can only be Setext
	if n.Lines().Len() > 1 {
		return r.renderSetextHeading(n, entering)
	}
	// Otherwise it's up to the configuration
	if r.config.IsSetext() {
		return r.renderSetextHeading(n, entering)
	}
	return r.renderATXHeading(n, entering)
}

func (r *Renderer) renderATXHeading(node *ast.Heading, entering bool) ast.WalkStatus {
	if entering {
		r.rc.writer.WriteBytes(bytes.Repeat([]byte("#"), node.Level))
		// Only print space after heading if non-empty
		if node.HasChildren() {
			r.rc.writer.WriteBytes([]byte(" "))
		}
	} else {
		if r.config.HeadingStyle == HeadingStyleATXSurround {
			r.rc.writer.WriteBytes([]byte(" "))
			r.rc.writer.WriteBytes(bytes.Repeat([]byte("#"), node.Level))
		}
	}
	return ast.WalkContinue
}

func (r *Renderer) renderSetextHeading(node *ast.Heading, entering bool) ast.WalkStatus {
	if entering {
		return ast.WalkContinue
	}
	underlineChar := [...][]byte{[]byte(""), []byte("="), []byte("-")}[node.Level]
	underlineWidth := 3
	if r.config.HeadingStyle == HeadingStyleFullWidthSetext {
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			lineWidth := line.Len()

			if lineWidth > underlineWidth {
				underlineWidth = lineWidth
			}
		}
	}
	r.rc.writer.WriteBytes([]byte("\n"))
	r.rc.writer.WriteBytes(bytes.Repeat(underlineChar, underlineWidth))
	return ast.WalkContinue
}

func (r *Renderer) renderThematicBreak(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		breakChars := []byte{'-', '*', '_'}
		breakChar := breakChars[r.config.ThematicBreakStyle : r.config.ThematicBreakStyle+1]
		breakLen := int(max(r.config.ThematicBreakLength, ThematicBreakLengthMinimum))
		r.rc.writer.WriteBytes(bytes.Repeat(breakChar, breakLen))
	}
	return ast.WalkContinue
}

func (r *Renderer) renderCodeBlock(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.rc.writer.PushPrefix(r.config.Bytes())
		// Skip translation for code block content
		r.rc.skipTranslation = true
		r.renderLines(node, entering)
	} else {
		r.rc.writer.PopPrefix()
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderFencedCodeBlock(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.FencedCodeBlock)
	r.rc.writer.WriteBytes([]byte("```"))
	if entering {
		r.rc.skipTranslation = true
		if info := n.Info; info != nil {
			r.rc.writer.WriteBytes(info.Value(r.rc.source))
		}
		r.rc.writer.FlushLine()
		r.renderLines(node, entering)
	} else {
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderHTMLBlock(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.HTMLBlock)
	if entering {
		if r.config.TextTransformer != nil {
			// Collect all HTML block content into a single string
			var htmlContent strings.Builder
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				htmlContent.Write(line.Value(r.rc.source))
			}

			// Add closure line if present
			if n.HasClosure() {
				htmlContent.Write(n.ClosureLine.Value(r.rc.source))
			}

			// Send the entire HTML content to the TextTransformer
			htmlStr := htmlContent.String()
			if translation, ok := r.config.TextTransformer.Transform(TextTypeHTML, htmlStr); ok {
				// Write the translated HTML directly
				r.rc.writer.WriteBytes([]byte(translation))
				return ast.WalkContinue
			}
		}

		// Fall back to default behavior if no transformation happened
		r.rc.skipTranslation = true
		r.renderLines(node, entering)
	} else {
		if n.HasClosure() {
			r.rc.writer.WriteLine(n.ClosureLine.Value(r.rc.source))
		}
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderList(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		n := node.(*ast.List)
		r.rc.lists = append(r.rc.lists, listContext{
			list: n,
			num:  n.Start,
		})
	} else {
		r.rc.lists = r.rc.lists[:len(r.rc.lists)-1]
	}
	return ast.WalkContinue
}

func (r *Renderer) renderListItem(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		var itemPrefix []byte
		l := r.rc.lists[len(r.rc.lists)-1]

		if l.list.IsOrdered() {
			itemPrefix = append(itemPrefix, []byte(fmt.Sprint(l.num))...)
			r.rc.lists[len(r.rc.lists)-1].num += 1
		}
		itemPrefix = append(itemPrefix, l.list.Marker, ' ')
		// Prefix the current line with the item prefix
		r.rc.writer.PushPrefix(itemPrefix, 0, 0)
		// Prefix subsequent lines with padding the same length as the item prefix
		indentLen := int(max(r.config.NestedListLength, NestedListLengthMinimum))
		indent := bytes.Repeat([]byte{' '}, indentLen)
		r.rc.writer.PushPrefix(bytes.Repeat(indent, len(itemPrefix)), 1)
	} else {
		r.rc.writer.PopPrefix()
		r.rc.writer.PopPrefix()
	}
	return ast.WalkContinue
}

// inline html tags
func (r *Renderer) renderRawHTML(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.RawHTML)
	if entering {
		if r.config.TextTransformer != nil {
			// For RawHTML, we just process this single node
			// We'll capture the complete HTML structure during translation step
			// with a custom TextTransformer approach
			var htmlContent strings.Builder
			segments := n.Segments
			for i := 0; i < segments.Len(); i++ {
				segment := segments.At(i)
				htmlContent.Write(segment.Value(r.rc.source))
			}

			// Send the HTML content to the TextTransformer
			htmlStr := htmlContent.String()
			if translation, ok := r.config.TextTransformer.Transform(TextTypeHTML, htmlStr); ok {
				// Write the translated HTML directly
				r.rc.writer.WriteBytes([]byte(translation))
				return ast.WalkContinue
			}
		}

		// Fall back to default behavior if no transformation happened
		r.renderSegments(n.Segments, false)
	}
	return ast.WalkContinue
}

func (r *Renderer) renderText(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.Text)

	if entering {
		text := n.Value(r.rc.source)
		nextIsSibling := node.NextSibling() != nil && node.NextSibling().Kind() == ast.KindText

		// Initialize or append to text buffer in renderContext
		if !r.rc.textBufferActive {
			// Initialize buffer
			if r.rc.textBuffer == nil {
				r.rc.textBuffer = &bytes.Buffer{}
			} else {
				r.rc.textBuffer.Reset()
			}
			r.rc.textBuffer.Write(text)
			r.rc.textBufferActive = true
			// Store this node's line break status
			if n.SoftLineBreak() {
				r.rc.pendingLineBreaks = append(r.rc.pendingLineBreaks, true)
			}
		} else {
			// If we have pending line breaks from previous Text nodes, add them now
			if len(r.rc.pendingLineBreaks) > 0 {
				for _, hasBreak := range r.rc.pendingLineBreaks {
					if hasBreak {
						// Add a newline character to represent the line break
						r.rc.textBuffer.WriteByte('\n')
					}
				}
				// Clear pending breaks
				r.rc.pendingLineBreaks = r.rc.pendingLineBreaks[:0]
			}

			// Append current text
			r.rc.textBuffer.Write(text)

			// Store this node's line break status
			if n.SoftLineBreak() {
				r.rc.pendingLineBreaks = append(r.rc.pendingLineBreaks, true)
			}
		}

		// If this is the last Text node in a sequence, process all accumulated text
		if !nextIsSibling {
			textStr := r.rc.textBuffer.String()

			// Check if we have a translation for this text
			if r.config.TextTransformer != nil && !r.rc.skipTranslation {
				trimmedText := strings.TrimSpace(textStr)

				if translation, ok := r.config.TextTransformer.Transform(TextTypePlain, trimmedText); ok {
					// Preserve the original leading and trailing spaces
					leadingSpaces := textStr[:len(textStr)-len(strings.TrimLeftFunc(textStr, unicode.IsSpace))]
					trailingSpaces := textStr[len(strings.TrimRightFunc(textStr, unicode.IsSpace)):]

					// Apply translation with preserved spaces
					textStr = leadingSpaces + translation + trailingSpaces
				}
			}

			// Write the accumulated text
			r.rc.writer.WriteBytes([]byte(textStr))

			// Handle final node's line break if needed
			lastNodeHasLineBreak := len(r.rc.pendingLineBreaks) > 0 && r.rc.pendingLineBreaks[len(r.rc.pendingLineBreaks)-1]
			if lastNodeHasLineBreak {
				r.rc.writer.EndLine()
			}

			// Reset text buffer state
			r.rc.textBufferActive = false
			r.rc.pendingLineBreaks = nil
		}
	}

	return ast.WalkContinue
}

func (r *Renderer) renderSegments(segments *text.Segments, asLines bool) {
	for i := 0; i < segments.Len(); i++ {
		segment := segments.At(i)
		value := segment.Value(r.rc.source)
		r.rc.writer.WriteBytes(value)
		if asLines {
			r.rc.writer.FlushLine()
		}
	}
}

func (r *Renderer) renderLines(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		lines := node.Lines()
		r.renderSegments(lines, true)
	}
	return ast.WalkContinue
}

func (r *Renderer) renderLink(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.Link)
	if entering {
		r.rc.writer.WriteBytes([]byte("["))
		// Text content should be translated, skipTranslation is false by default
	} else {
		// Only set skipTranslation when rendering the URL part
		r.rc.skipTranslation = true
		r.rc.writer.WriteBytes([]byte("]("))
		r.rc.writer.WriteBytes(n.Destination)
		if len(n.Title) > 0 {
			r.rc.writer.WriteBytes([]byte(" \""))
			r.rc.writer.WriteBytes(n.Title)
			r.rc.writer.WriteBytes([]byte("\""))
		}
		r.rc.writer.WriteBytes([]byte(")"))
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderImage(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.Image)
	if entering {
		r.rc.writer.WriteBytes([]byte("!["))
		// Alt text should be translated, skipTranslation is false by default
	} else {
		// Only set skipTranslation when rendering the URL part
		r.rc.skipTranslation = true
		r.rc.writer.WriteBytes([]byte("]("))
		r.rc.writer.WriteBytes(n.Destination)
		if len(n.Title) > 0 {
			r.rc.writer.WriteBytes([]byte(" \""))
			// Temporarily disable skipTranslation to allow the title to be translated
			r.rc.skipTranslation = false
			r.rc.writer.WriteBytes(n.Title)
			// Re-enable skipTranslation for the rest of the URL
			r.rc.skipTranslation = true
			r.rc.writer.WriteBytes([]byte("\""))
		}
		r.rc.writer.WriteBytes([]byte(")"))
		r.rc.skipTranslation = false
	}
	return ast.WalkContinue
}

func (r *Renderer) renderCodeSpan(node ast.Node, entering bool) ast.WalkStatus {
	if entering {
		r.rc.skipTranslation = true
		// get contents of codespan
		var contentBytes []byte
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			text := c.(*ast.Text).Segment
			contentBytes = append(contentBytes, text.Value(r.rc.source)...)
		}
		contents := string(contentBytes)

		//
		var beginsWithSpace bool
		var endsWithSpace bool
		var beginsWithBackTick bool
		var endsWithBackTick bool
		isOnlySpace := true
		backtickLengths := []int{}
		count := 0
		for i, c := range contents {
			if i == 0 {
				beginsWithSpace = unicode.IsSpace(c)
				beginsWithBackTick = c == '`'
			} else if i == len(contents)-1 {
				endsWithSpace = unicode.IsSpace(c)
				endsWithBackTick = c == '`'
			}
			if !unicode.IsSpace(c) {
				isOnlySpace = false
			}
			if c == '`' {
				count++
			} else if count > 0 {
				backtickLengths = append(backtickLengths, count)
				count = 0
			}
		}
		if count > 0 {
			backtickLengths = append(backtickLengths, count)
		}

		// Surround the codespan with the minimum number of backticks required to contain the span.
		for i := 1; i <= len(contentBytes); i++ {
			if !slices.Contains(backtickLengths, i) {
				r.rc.codeSpanContext.backtickLength = i
				break
			}
		}
		r.rc.writer.WriteBytes(bytes.Repeat([]byte("`"), r.rc.codeSpanContext.backtickLength))

		// Check if the code span needs to be padded with spaces
		if beginsWithSpace && endsWithSpace && !isOnlySpace || beginsWithBackTick || endsWithBackTick {
			r.rc.codeSpanContext.padSpace = true
			r.rc.writer.WriteBytes([]byte(" "))
		}
	} else {
		if r.rc.codeSpanContext.padSpace {
			r.rc.writer.WriteBytes([]byte(" "))
		}
		r.rc.writer.WriteBytes(bytes.Repeat([]byte("`"), r.rc.codeSpanContext.backtickLength))
		r.rc.skipTranslation = false
	}

	return ast.WalkContinue
}

func (r *Renderer) renderEmphasis(node ast.Node, entering bool) ast.WalkStatus {
	n := node.(*ast.Emphasis)
	r.rc.writer.WriteBytes(bytes.Repeat([]byte{'*'}, n.Level))
	return ast.WalkContinue
}

// Table rendering functions
func (r *Renderer) renderTable(
	w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	// Tables are rendered as markdown tables with | separators
	if !entering {
		// End the table with a newline
		// r.rc.writer.EndLine()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTableHeader(
	w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.rc.writer.WriteBytes([]byte("|"))
	} else {
		// After rendering all header cells, add the separator row
		r.rc.writer.EndLine()

		tableNode := n.Parent()
		alignments := tableNode.(*east.Table).Alignments

		r.rc.writer.WriteByte('|')
		for _, alignment := range alignments {
			r.rc.writer.WriteByte(' ')
			switch alignment {
			case east.AlignLeft:
				r.rc.writer.WriteBytes([]byte(":----- "))
			case east.AlignRight:
				r.rc.writer.WriteBytes([]byte("-----: "))
			case east.AlignCenter:
				r.rc.writer.WriteBytes([]byte(":----: "))
			default:
				r.rc.writer.WriteBytes([]byte("----- "))
			}
			r.rc.writer.WriteByte('|')
		}
		r.rc.writer.EndLine()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTableRow(
	w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Start the row with a pipe
		r.rc.writer.WriteByte('|')
	} else {
		// End the row with a pipe and a newline
		r.rc.writer.EndLine()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTableCell(
	w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Add a space after the pipe for readability
		r.rc.writer.WriteByte(' ')
	} else {
		// Add a space and pipe after each cell
		r.rc.writer.WriteBytes([]byte(" |"))
	}
	return ast.WalkContinue, nil
}

type renderContext struct {
	writer *markdownWriter
	// source is the markdown source
	source []byte
	// listMarkers is the marker character used for the current list
	lists           []listContext
	codeSpanContext codeSpanContext
	// skipTranslation indicates whether we're inside a node type that shouldn't be translated
	skipTranslation bool
	// Text accumulation fields
	textBuffer        *bytes.Buffer
	textBufferActive  bool
	pendingLineBreaks []bool
}

type listContext struct {
	list *ast.List
	num  int
}

// codeSpanContext holds state about how the current codespan should be rendererd.
type codeSpanContext struct {
	// number of backticks to use
	backtickLength int
	// whether to surround the codespan with spaces
	padSpace bool
}

// newRenderContext returns a new renderContext object
func newRenderContext(writer io.Writer, source []byte, config *Config) renderContext {
	return renderContext{
		writer: newMarkdownWriter(writer, config),
		source: source,
	}
}
