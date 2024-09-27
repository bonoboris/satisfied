package app

import (
	"strconv"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/math32"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var grid = Grid{SnapStep: 1}

type Grid struct {
	// SnapStep is the snap step in world units, 0 to disable
	SnapStep float32
}

// Snap returns the vector snapped to the grid
func (g Grid) Snap(v rl.Vector2) rl.Vector2 {
	if g.SnapStep == 0 {
		return v
	}
	return vec2(g.SnapStep*math32.Round(v.X)/g.SnapStep, g.SnapStep*math32.Round(v.Y)/g.SnapStep)
}

// Draw grid
func (g Grid) Draw() {
	s := dims.World.TopLeft()
	e := dims.World.BottomRight()

	zoom := camera.Zoom()
	px := 1 / zoom // 1 pixel in worl units

	// 2x2 dasehd lines (same as in blueprint)
	if zoom > 16. {
		x0 := math32.Ceil(s.X/2) * 2
		y0 := math32.Ceil(s.Y/2) * 2
		for x := x0; x < e.X; x += 2 {
			for y := y0; y < e.Y; y += 2 {
				v := vec2(x, y)
				rl.DrawLineEx(v.Add(vec2(0, -0.25)), v.Add(vec2(0, 0.25)), px, colors.Gray300)
				rl.DrawLineEx(v.Add(vec2(-0.25, 0)), v.Add(vec2(0.25, 0)), px, colors.Gray300)
			}
		}
	}

	// 4x4 lines (half foundation size)
	if zoom > 8. {
		x0 := math32.Ceil(s.X/4) * 4
		y0 := math32.Ceil(s.Y/4) * 4
		for x := x0; x < e.X; x += 4 {
			rl.DrawLineEx(vec2(x, s.Y), vec2(x, e.Y), px, colors.Gray300)
		}

		for y := y0; y < e.Y; y += 4 {
			rl.DrawLineEx(vec2(s.X, y), vec2(e.X, y), px, colors.Gray300)
		}
	}

	x0 := math32.Ceil(s.X/8) * 8
	y0 := math32.Ceil(s.Y/8) * 8
	// 8x8 lines (foundation size)
	for x := x0; x < e.X; x += 8 {
		rl.DrawLineEx(vec2(x, s.Y), vec2(x, e.Y), 1.*px, colors.Gray500)
	}

	for y := y0; y < e.Y; y += 8 {
		rl.DrawLineEx(vec2(s.X, y), vec2(e.X, y), 1.*px, colors.Gray500)
	}

	// orgin lines
	if s.X <= 0 && e.X >= 0 {
		rl.DrawLineEx(vec2(0, s.Y), vec2(0, e.Y), 2.*px, colors.Blue500)
	}
	if s.Y <= 0 && e.Y >= 0 {
		rl.DrawLineEx(vec2(s.X, 0), vec2(e.X, 0), 2.*px, colors.Blue500)
	}

	// mouse lines
	if mouse.InScene {
		c := colors.WithAlpha(colors.Green500, 0.5)
		pos := mouse.SnappedPos
		if camera.Zooming {
			pos = camera.WorldPos(camera.ZoomAt)
		}
		rl.DrawLineEx(vec2(pos.X, s.Y), vec2(pos.X, e.Y), 2.*px, c)
		rl.DrawLineEx(vec2(s.X, pos.Y), vec2(e.X, pos.Y), 2.*px, c)
	}

	// labels

	step := float32(8) // each foundation
	if zoom < 4. {
		step *= 2 // each 2 foundations
	}
	if zoom < 2. {
		step *= 2 // each 4 foundations
	}
	off := 5 * px

	x0 = math32.Ceil(s.X/step) * step
	y0 = math32.Ceil(s.Y/step) * step

	fontSize := 16 * px
	for x := x0; x < e.X; x += step {
		var txt string
		if settings.TickFoundation {
			txt = strconv.FormatFloat(float64(x/8), 'f', 0, 32)
		} else {
			txt = strconv.FormatFloat(float64(x), 'f', 0, 32)
		}
		rl.DrawTextEx(font, txt, vec2(x+off, s.Y+off), fontSize, 0, colors.Gray700)
	}
	for y := y0; y < e.Y; y += step {
		var txt string
		if settings.TickFoundation {
			txt = strconv.FormatFloat(float64(y/8), 'f', 0, 32)
		} else {
			txt = strconv.FormatFloat(float64(y), 'f', 0, 32)
		}
		rl.DrawTextEx(font, txt, vec2(s.X+off, y+off), fontSize, 0, colors.Gray700)
	}
}
