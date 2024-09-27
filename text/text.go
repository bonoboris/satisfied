// package text - Implements custom text drawing functions

package text

import (
	"fmt"
	"math"
	"sort"
	"unicode/utf8"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/log"
	rl "github.com/gen2brain/raylib-go/raylib"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// Enums
////////////////////////////////////////////////////////////////////////////////////////////////////

type Align int

const (
	AlignStart Align = iota
	AlignMiddle
	AlignEnd
)

func (a Align) String() string {
	switch a {
	case AlignStart:
		return "AlignStart"
	case AlignMiddle:
		return "AlignMiddle"
	case AlignEnd:
		return "AlignEnd"
	default:
		return "invalid"
	}
}

type Wrap int

const (
	// No wrapping (X-overflow possible)
	WrapNone Wrap = iota
	// Wrap on words only (X-overflow possible)
	WrapWord
	// Wrap on words then on chars (no X-overflow)
	WrapChar
)

func (w Wrap) String() string {
	switch w {
	case WrapNone:
		return "WrapNone"
	case WrapWord:
		return "WrapWord"
	case WrapChar:
		return "WrapChar"
	default:
		return "invalid"
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// DrawText
////////////////////////////////////////////////////////////////////////////////////////////////////

type Options struct {
	// Font to use
	Font rl.Font
	// Font size
	Size float32
	// Text color
	Color rl.Color
	// Spacing between letters
	Spacing float32
	// Spacing between lines
	LineSpacing float32
	// Whether to print/draw debug info
	Debug bool

	// Horizontal alignment
	Align Align
	// Vertical alignment
	VerticalAlign Align
	// Whether to wrap text
	Wrap Wrap
}

func (opts Options) getYOff(bounds rl.Rectangle, numLines int) float32 {
	n := float32(numLines)
	height := n*(opts.Size+opts.LineSpacing) - opts.LineSpacing
	if height >= bounds.Height {
		return 0
	}
	switch opts.Align {
	case AlignStart:
		return 0
	case AlignMiddle:
		return (bounds.Height - height) / 2
	case AlignEnd:
		return bounds.Height - height
	default:
		panic("invalid align")
	}
}

func (opts Options) getX(bounds rl.Rectangle, textWidth float32) float32 {
	if textWidth >= bounds.Width {
		return bounds.X
	}
	switch opts.Align {
	case AlignStart:
		return bounds.X
	case AlignMiddle:
		return bounds.X + (bounds.Width-textWidth)/2
	case AlignEnd:
		return bounds.X + bounds.Width - textWidth
	default:
		panic("invalid align")
	}
}

func (opts Options) draw(text string, pos rl.Vector2) {
	rl.DrawTextEx(opts.Font, text, pos, opts.Size, opts.Spacing, opts.Color)
}

// DrawText draws the text in the given bounds.
func DrawText(bounds rl.Rectangle, text string, opts Options) {
	// TODO: Optimize: no new alloc, re implement MeasureTextEx, draw char by char instead of line by line
	if opts.Debug {
		fmt.Printf("DrawText: bounds=%v text=%v opts=%+v\n", bounds, text, opts)
	}

	lines := getLines(text, bounds.Width, opts.Font, opts.Size, opts.Spacing, opts.Wrap)
	yOff := opts.getYOff(bounds, len(lines))
	pos := bounds.TopLeft().Add(rl.Vector2{X: 0, Y: yOff})
	if opts.Debug {
		fmt.Printf("yOff=%v\n", yOff)
	}
	for _, line := range lines {
		if opts.Debug {
			fmt.Printf("line=%+v\n", line)
		}
		if pos.Y+opts.Size > bounds.Y+bounds.Height {
			// truncate on Y
			if opts.Debug {
				fmt.Printf("truncated on Y\n")
			}
			break
		}
		pos.X = opts.getX(bounds, line.width)
		if opts.Debug {
			fmt.Printf("print %v at pos=%v\n", line, pos)
		}
		opts.draw(text[line.start:line.renderEnd], pos)
		pos.Y += opts.Size + opts.LineSpacing
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Area
////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	areaPadding = 3
	cursorWidth = 3
)

type AreaOptions struct {
	// Font to use
	Font rl.Font
	// Font size
	Size float32
	// Text color
	Color rl.Color
	// Spacing between letters
	Spacing float32
	// Spacing between lines
	LineSpacing float32
	// Draw in disabled state and ignore inputs
	Disabled bool
}

func (opts AreaOptions) getWidth(text string) float32 {
	return rl.MeasureTextEx(opts.Font, text, opts.Size, opts.Spacing).X
}

func (opts AreaOptions) draw(text string, pos rl.Vector2) {
	color := opts.Color
	if opts.Disabled {
		color = colors.Gray500
	}
	rl.DrawTextEx(opts.Font, text, pos, opts.Size, opts.Spacing, color)
}

// Area represents a text area.
//
// It retains its state.
//
// TODO:
// - Selection
// - Copy / paste
// - Undo / redo
// - Scrollbar and scroll with mouse wheel
// - auto grow with max height
type Area struct {
	bounds    rl.Rectangle
	text      string // TODO: []byte, but this requires a rl.DrawText func that takes []byte
	opts      AreaOptions
	focused   bool
	cursor    int
	scroll    float32
	lines     []line
	lastPress string
}

// NewArea returns a new, unfocused, text [Area] with the given bounds, text and text options
//
// It works best with a monospaced font.
func NewArea(bounds rl.Rectangle, text string, opts AreaOptions) Area {
	// ensure at least 1 line
	log.Debug("text.NewArea", "bounds", bounds, "text", text, "opts", opts)
	bounds.Height = max(bounds.Height, opts.Size+opts.LineSpacing+2*areaPadding)
	a := Area{bounds: bounds, text: text, opts: opts}
	a.recomputeLines()
	a.traceState("after", "NewArea")
	return a
}

// Focused returns whether the area is focused
func (a *Area) Focused() bool { return a.focused }

// Text returns the current text in the area
func (a *Area) Text() string { return a.text }

// SetFocused sets the focused state of the area
func (a *Area) SetFocused(focused bool) {
	if a.focused != focused {
		log.Debug("text.Area.SetFocused", "newValue", focused)
		a.focused = focused
		a.recomputeLines()
		a.ensureCursorInView()
	}
}

// SetText sets the text in the area
func (a *Area) SetText(text string) {
	if a.text != text {
		log.Debug("text.Area.SetText", "newValue", text)
		a.text = text
		a.recomputeLines()
		a.cursor = 0
		a.ensureCursorInView()
	}
}

// SetBounds sets the bounds of the area
func (a *Area) SetBounds(bounds rl.Rectangle) {
	// ensure at least 1 line
	bounds.Height = max(bounds.Height, a.opts.Size+a.opts.LineSpacing+2*areaPadding)

	if a.bounds != bounds {
		log.Debug("text.Area.SetBounds", "newValue", bounds)
		a.bounds = bounds
		a.recomputeLines()
		a.ensureCursorInView()
	}
}

func (a *Area) SetDisabled(disabled bool) {
	if disabled {
		a.focused = false
	}
	a.opts.Disabled = disabled
}

// Draw draws the text area
func (a *Area) Draw(keyPressed int32) {
	// draw disabled and return
	if a.opts.Disabled {
		rl.DrawRectangleRec(a.bounds, colors.Gray100)
		rl.DrawRectangleLinesEx(a.bounds, 1, colors.Gray500)
		a.drawText()
		return
	}

	lmb := rl.IsMouseButtonPressed(rl.MouseButtonLeft)
	pos := rl.GetMousePosition()
	mouseIn := a.bounds.CheckCollisionPoint(pos)
	isCtrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	// update focus
	if lmb {
		a.SetFocused(mouseIn)
		if a.focused {
			a.setCursorFromRenderPos(pos)
		}
	}
	if keyPressed == rl.KeyEscape {
		a.SetFocused(false)
	}

	// draw unfocused and return
	if !a.focused {
		rl.DrawRectangleRec(a.bounds, colors.White)
		if mouseIn {
			rl.DrawRectangleLinesEx(a.bounds, 1, colors.Blue300)
		} else {
			rl.DrawRectangleLinesEx(a.bounds, 1, colors.Gray500)
		}
		a.drawText()
		return
	}

	// update cursor / text
	switch keyPressed {
	case rl.KeyLeft:
		a.lastPress = "ArrowLeft"
		a.cursor -= a.backwardOffset(isCtrl)
		a.ensureCursorInView()
	case rl.KeyRight:
		a.lastPress = "ArrowRight"
		a.cursor += a.forwardOffset(isCtrl)
		a.ensureCursorInView()
	case rl.KeyUp:
		a.lastPress = "ArrowUp"
		row, col := a.cursorPos()
		a.setCursorFromPos(row-1, col)
		a.ensureCursorInView()
	case rl.KeyDown:
		a.lastPress = "ArrowDown"
		row, col := a.cursorPos()
		a.setCursorFromPos(row+1, col)
		a.ensureCursorInView()
	case rl.KeyHome:
		a.lastPress = "Home"
		a.cursor = 0
		a.ensureCursorInView()
	case rl.KeyEnd:
		a.lastPress = "End"
		a.cursor = len(a.text)
		a.ensureCursorInView()
	case rl.KeyBackspace:
		a.lastPress = "Backspace"
		off := a.backwardOffset(isCtrl)
		if off > 0 {
			a.text = a.text[:a.cursor-off] + a.text[a.cursor:]
			a.recomputeLines()
			a.cursor -= off
			a.ensureCursorInView()
		}
	case rl.KeyDelete:
		a.lastPress = "Delete"
		off := a.forwardOffset(isCtrl)
		if off > 0 {
			a.text = a.text[:a.cursor] + a.text[a.cursor+off:]
			a.recomputeLines()
			a.ensureCursorInView()
		}
	case rl.KeyEnter:
		a.lastPress = "Enter"
		if !isCtrl {
			a.text = a.text[:a.cursor] + "\n" + a.text[a.cursor:]
			a.recomputeLines()
			a.cursor++
			a.ensureCursorInView()
		}
	default:
		if r := rl.GetCharPressed(); r != 0 {
			a.lastPress = string(r)
			a.text = a.text[:a.cursor] + string(r) + a.text[a.cursor:]
			a.recomputeLines()
			a.cursor++
			a.ensureCursorInView()
		}
	}

	// Draw
	rl.DrawRectangleRec(a.bounds, colors.White)
	rl.DrawRectangleLinesEx(a.bounds, 1, colors.Blue500)
	a.drawText()
	start := a.cursorRenderPos()
	end := start.Add(rl.Vector2{X: 0, Y: a.opts.Size})
	rl.DrawLineEx(start, end, cursorWidth, colors.Blue500)
}

func (a *Area) traceState(key, val string) {
	log.Trace("text.Area", key, val, "text", a.text, "bounds", a.bounds, "lines", a.lines, "cursor", a.cursor, "scroll", a.scroll, "focused", a.focused, "lastPress", a.lastPress)
}

func (a *Area) ensureCursorInView() {
	bounds := a.textBounds()
	ystart := a.cursorRenderPos().Y
	yend := ystart + a.opts.Size

	if ystart < bounds.Y {
		a.scroll -= bounds.Y - ystart
	} else if yend > bounds.Y+bounds.Height {
		a.scroll += yend - (bounds.Y + bounds.Height)
	}
}

func (a *Area) drawText() {
	bounds := a.textBounds()
	pos := bounds.TopLeft()
	lineHeight := a.opts.Size + a.opts.LineSpacing
	nSkip := int(math.Ceil(float64(a.scroll) / float64(lineHeight)))

	pos.Y += float32(nSkip)*lineHeight - a.scroll
	// start Y should be somewhere between the text bounds top and 1 lineHeight below
	assert(pos.Y >= bounds.Y, "pos.Y < bounds.Y")
	assert(pos.Y <= bounds.Y+lineHeight, "pos.Y > bounds.Y + lineHeight")
	for _, line := range a.lines[nSkip:] {
		if pos.Y+a.opts.Size > bounds.Y+bounds.Height {
			// truncate on Y
			break
		}
		a.opts.draw(a.text[line.start:line.renderEnd], pos)
		pos.Y += lineHeight
	}
}

func (a *Area) forwardOffset(nextWord bool) int {
	if a.cursor == len(a.text) {
		log.Debug("text.Area.forwardOffset", "offset", 0, "reason", "cursor is at the end")
		return 0
	}
	if !nextWord {
		log.Debug("text.Area.forwardOffset", "offset", 1, "reason", "next char mode")
		return 1
	}
	hasSpace := false
	for i, r := range a.text[a.cursor:] {
		switch r {
		case '\n':
			if i == 0 {
				log.Debug("text.Area.forwardOffset", "offset", 1, "reason", "cursor was just before a newline")
				return 1
			} else {
				log.Debug("text.Area.forwardOffset", "offset", i, "reason", "stop at newline")
				return i
			}
		case ' ':
			hasSpace = true
		default:
			if hasSpace {
				log.Debug("text.Area.forwardOffset", "offset", i, "reason", "stop at next word")
				return i
			}
		}
	}
	offset := len(a.text) - a.cursor
	log.Debug("text.Area.forwardOffset", "offset", offset, "reason", "stop at the end")
	return offset
}

func (a *Area) backwardOffset(nextWord bool) int {
	if a.cursor == 0 {
		log.Debug("text.Area.backwardOffset", "offset", 0, "reason", "cursor is at the start")
		return 0
	}
	if !nextWord {
		log.Debug("text.Area.backwardOffset", "offset", 1, "reason", "previous char mode")
		return 1
	}

	hasSpace := false
	for k := a.cursor; k > 0; {
		r, l := utf8.DecodeLastRuneInString(a.text[:k])
		switch r {
		case '\n':
			offset := a.cursor - k
			if offset == 0 {
				log.Debug("text.Area.backwardOffset", "offset", 1, "reason", "cursor was just after a newline")
				return 1
			} else {
				log.Debug("text.Area.backwardOffset", "offset", offset, "reason", "stop at newline")
				return offset
			}
		case ' ':
			hasSpace = true
		default:
			if hasSpace {
				offset := a.cursor - k
				log.Debug("text.Area.backwardOffset", "offset", offset, "reason", "stop at previous word end")
				// stops after the first non-space before one or more spaces
				return a.cursor - k
			}
		}
		k -= l
	}
	log.Debug("text.Area.backwardOffset", "offset", a.cursor, "reason", "stop at the start")
	return a.cursor
}

func (a *Area) textBounds() rl.Rectangle {
	return rl.NewRectangleV(
		a.bounds.Position().AddValue(areaPadding),
		a.bounds.Size().SubtractValue(2*areaPadding))
}

func (a *Area) cursorPos() (row int, col int) {
	row = sort.Search(len(a.lines), func(i int) bool { return a.lines[i].end > a.cursor })
	if row == len(a.lines) {
		row--
	}
	l := a.lines[row]
	col = a.cursor - l.start
	assert(row >= 0, "cursor row < 0")
	assert(row < len(a.lines), "cursor row >= len(a.lines)")
	assert(col >= 0, "cursor col < 0")
	return row, col
}

func (a *Area) setCursorFromPos(row int, col int) {
	defer func() {
		log.Debug("text.Area.setCursorFrom", "row", row, "col", col, "newCursor", a.cursor)
	}()
	if row < 0 {
		a.cursor = 0
	} else if row >= len(a.lines) {
		a.cursor = len(a.text)
	} else {
		l := a.lines[row]
		a.cursor = a.lines[row].start + max(0, min(col, l.renderEnd-l.start-1))
	}
}

func (a *Area) cursorRenderPos() (pos rl.Vector2) {
	row, _ := a.cursorPos()
	tl := a.textBounds().TopLeft()
	yOff := float32(row) * (a.opts.Size + a.opts.LineSpacing)
	xOff := a.opts.getWidth(a.text[a.lines[row].start:a.cursor])
	return tl.Add(rl.Vector2{X: xOff, Y: yOff - a.scroll})
}

func (a *Area) setCursorFromRenderPos(pos rl.Vector2) {
	log.Debug("text.Area.setCursorFrom", "x", pos.X, "y", pos.Y)
	off := pos.Subtract(a.textBounds().TopLeft())
	off.Y += a.scroll
	lineHeight := a.opts.Size + a.opts.LineSpacing
	log.Debug("text.Area.setCursorFrom", "offsetX", off.X, "offsetY", off.Y, "lineHeight", lineHeight)
	row := int((off.Y) / lineHeight)
	if row < 0 {
		log.Debug("text.Area.setCursorFrom", "newCursor", 0, "reason", "row < 0")
		a.cursor = 0
		return
	} else if row >= len(a.lines) {
		log.Debug("text.Area.setCursorFrom", "newCursor", len(a.text), "reason", "row >= num_lines")
		a.cursor = len(a.text)
		return
	}
	l := a.lines[row]
	if l.start == l.renderEnd {
		log.Debug("text.Area.setCursorFrom", "row", row, "newCursor", l.start, "reason", "clicked on empty line")
		a.cursor = l.start
		return
	}
	if off.X < 0 {
		log.Debug("text.Area.setCursorFrom", "row", row, "newCursor", l.start, "reason", "clicked before first line char")
		a.cursor = l.start
		return
	}

	var prevW float32
	// TODO: We could use a binary search here + another getWidth call to compare closest char
	for i := range l.renderEnd - l.start - 1 {
		w := a.opts.getWidth(a.text[l.start : l.start+i+1])
		if w >= off.X {
			if off.X-prevW < w-off.X {
				log.Debug("text.Area.setCursorFrom", "row", row, "col", i, "newCursor", l.start+i, "reason", "clicked closer to this char start")
				a.cursor = l.start + i
				return
			} else {
				log.Debug("text.Area.setCursorFrom", "row", row, "col", i+1, "newCursor", l.start+i+1, "reason", "clicked closer the previous char end")
				a.cursor = l.start + i + 1
				return
			}
		}
		prevW = w
	}
	col := l.renderEnd - l.start - 1
	if l.endType == textEnd {
		col++
	}
	log.Debug("text.Area.setCursorFrom", "row", row, "col", col, "newCursor", l.start+col, "reason", "clicked after last line char")
	a.cursor = l.start + col
}

func (a *Area) recomputeLines() {
	log.Debug("text.Area.recomputeLines")
	a.lines = getLines(a.text, a.textBounds().Width, a.opts.Font, a.opts.Size, a.opts.Spacing, WrapChar)
	a.traceState("after", "recomputeLines")
}

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////////////////////////

type lineEndType int

const (
	textEnd lineEndType = iota
	newLine
	spaceWrap
	charWrap
)

func (l lineEndType) String() string {
	switch l {
	case textEnd:
		return "textEnd"
	case newLine:
		return "newLine"
	case spaceWrap:
		return "spaceWrap"
	case charWrap:
		return "charWrap"
	default:
		return "invalid"
	}
}

type line struct {
	start, end, renderEnd int
	width                 float32
	endType               lineEndType
}

func (l line) String() string {
	return fmt.Sprintf("{s=%v e=%v r=%v w=%.1f eType=%v}", l.start, l.end, l.renderEnd, l.width, l.endType)
}

func getLines(text string, width float32, font rl.Font, size, spacing float32, wrap Wrap) []line {
	var lines []line
	start := 0
outer:
	for {
		renderEnd := start
		endSpace := -1
		var widthSpace float32
		// width from start to start+i excluded (in the loop)
		var curWidth float32
		for i, r := range text[start:] {
			nr := utf8.RuneLen(r)
			// width from start to start+i+nr included
			newWidth := rl.MeasureTextEx(font, text[start:start+i+nr], size, spacing).X
			if newWidth <= width {
				renderEnd += nr
			}
			if r == '\n' {
				lines = append(lines, line{
					start:     start,
					end:       start + i + nr,
					renderEnd: renderEnd,
					width:     newWidth,
					endType:   newLine,
				})
				start += i + nr
				continue outer
			}
			if wrap == WrapNone {
				curWidth = newWidth
				continue
			}
			if r == ' ' {
				endSpace = start + i + nr
				widthSpace = newWidth
			}
			if newWidth > width {
				if wrap == WrapChar && endSpace == -1 {
					// no space, break just before this char
					lines = append(lines, line{
						start:     start,
						end:       start + i,
						renderEnd: renderEnd,
						width:     curWidth,
						endType:   charWrap,
					})
					start += i
					continue outer
				} else if endSpace >= 0 {
					// break after last space
					lines = append(lines, line{
						start:     start,
						end:       endSpace,
						renderEnd: min(renderEnd, endSpace),
						width:     widthSpace,
						endType:   spaceWrap,
					})
					start = endSpace
					continue outer
				}
			}
			curWidth = newWidth
		}
		// all remaining chars fit
		lines = append(lines, line{
			start:     start,
			end:       len(text),
			renderEnd: renderEnd,
			width:     curWidth,
			endType:   textEnd,
		})
		break
	}
	return lines
}
