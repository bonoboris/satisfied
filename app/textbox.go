// text box - Define text boxes and handle [ModeEditTextBox]

package app

import (
	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/text"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	textBoxFontSize    = 24.    // text font size in pixels
	textBoxLineSpacing = -5.    // text line spacing in pixels
	textBoxMinSize     = 1      // minimum size of a text box in world units
	textBoxHandleSize  = 20     // size of the grab handle in pixels
	textBoxDefaultText = "Text" // default text box content
)

type TextBox struct {
	Bounds  rl.Rectangle
	Content string
}

func (tb TextBox) HandleRect() rl.Rectangle {
	br := tb.Bounds.BottomRight()
	size := textBoxHandleSize / camera.Zoom()
	return rl.NewRectangle(br.X-size, br.Y-size, size, size)
}

// Draw draws the textbox (must be called outside of Camera2D mode)
func (tb *TextBox) Draw(state DrawState, drawHandle bool) {
	if state == DrawSkip {
		return
	}
	if !dims.World.CheckCollisionRec(tb.Bounds) {
		return
	}
	color := state.transformColor(colors.WithAlpha(colors.Gray300, 0.5))
	rl.DrawRectangleRec(tb.Bounds, color)

	if state == DrawShadow {
		return
	}

	if drawHandle {
		// FIXME: this is a hacky way to draw the resize handle only when needed
		px := 1 / camera.Zoom()
		handle := tb.HandleRect()
		rl.DrawRectangleLinesEx(handle, 1*px, colors.Gray500)

		tl := handle.TopLeft()
		br := handle.BottomRight()
		bl := handle.BottomLeft()
		tr := handle.TopRight()

		ml := tl.Add(bl).Scale(0.5)
		mt := tl.Add(tr).Scale(0.5)
		mb := bl.Add(br).Scale(0.5)
		mr := tr.Add(br).Scale(0.5)

		rl.DrawLineEx(ml, mt, 1*px, colors.Gray500)
		rl.DrawLineEx(bl, tr, 1*px, colors.Gray500)
		rl.DrawLineEx(mb, mr, 1*px, colors.Gray500)
	}
	textOpts := text.Options{
		Font:          font,
		Size:          24 / camera.Zoom(),
		Color:         colors.Gray700,
		Align:         text.AlignMiddle,
		VerticalAlign: text.AlignMiddle,
		LineSpacing:   textBoxLineSpacing,
	}
	text.DrawText(tb.Bounds, tb.Content, textOpts)
}
