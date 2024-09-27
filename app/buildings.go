package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/matrix"
	"github.com/bonoboris/satisfied/text"
	rl "github.com/gen2brain/raylib-go/raylib"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// Building
////////////////////////////////////////////////////////////////////////////////////////////////////

type Building struct {
	DefIdx int
	Pos    rl.Vector2
	Rot    int32
}

func (b Building) String() string {
	if b.DefIdx == -1 {
		return fmt.Sprintf("%s{%v %v %d}", "<invalid>", b.Pos.X, b.Pos.Y, b.Rot)
	}
	return fmt.Sprintf("%s{%v %v %d}", b.Def().Class, b.Pos.X, b.Pos.Y, b.Rot)
}

func (b Building) Def() BuildingDef { return buildingDefs[b.DefIdx] }

func (b Building) matrix() matrix.Matrix {
	mid := grid.Snap(b.Def().Dims.Scale(0.5))
	return matrix.NewTranslateV(b.Pos).Rotate(b.Rot).TranslateV(mid.Negate())
}

func (b Building) Bounds() rl.Rectangle {
	dims := b.Def().Dims
	return b.matrix().ApplyRec(0, 0, dims.X, dims.Y)
}

func (b Building) DrawLabel(bounds rl.Rectangle) {
	bounds.X += 0.5
	bounds.Y += 0.5
	bounds.Width -= 1
	bounds.Height -= 1
	zoom := camera.Zoom()
	labelOpts := text.Options{
		Font:          labelFont,
		Size:          float32(settings.FontSize) / zoom,
		Color:         colors.Gray700,
		Align:         text.AlignMiddle,
		VerticalAlign: text.AlignMiddle,
	}
	text.DrawText(bounds, strings.ReplaceAll(b.Def().Class, " ", "\n"), labelOpts)
}

func (b Building) Draw(state DrawState) {
	if state == DrawSkip {
		return
	}
	mat := b.matrix()
	def := b.Def()
	bounds := mat.ApplyRec(0, 0, def.Dims.X, def.Dims.Y)

	if !dims.World.CheckCollisionRec(bounds) {
		// skip drawing if building is outside of the scene
		return
	}

	app.drawCounts.Buildings++
	rl.DrawRectangleRec(bounds, state.transformColor(colors.Blue300))

	if state == DrawShadow {
		return
	}

	rl.DrawRectangleLinesEx(bounds, 0.5, state.transformColor(colors.Blue500))

	for i := 0; i < def.BeltIn.len; i++ {
		def.BeltIn.arr[i].drawBeltIn(mat, state)
	}
	for i := 0; i < def.BeltOut.len; i++ {
		def.BeltOut.arr[i].drawBeltOut(mat, state)
	}
	for i := 0; i < def.PipeIn.len; i++ {
		def.PipeIn.arr[i].drawPipeIn(mat, state)
	}
	for i := 0; i < def.PipeOut.len; i++ {
		def.PipeOut.arr[i].drawPipeOut(mat, state)
	}
	b.DrawLabel(bounds)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// (Belt, Pipe)  input / output drawing
////////////////////////////////////////////////////////////////////////////////////////////////////

type inputOutput struct {
	Pos rl.Vector2
	Rot int32
}

func (io inputOutput) String() string {
	if io.Rot == 0 {
		return fmt.Sprintf("(%v,%v)", io.Pos.X, io.Pos.Y)
	} else {
		return fmt.Sprintf("(%v,%v, r=%d°)", io.Pos.X, io.Pos.Y, io.Rot)
	}
}

func (io inputOutput) matrix() matrix.Matrix {
	return matrix.NewTranslateV(io.Pos).Rotate(io.Rot)
}

func (inputOutput) drawTri(mat matrix.Matrix, x, y float32, c rl.Color) {
	rl.DrawTriangle(
		mat.Apply(x-0.25, y+0.25),
		mat.Apply(x+0.25, y+0.25),
		mat.Apply(x, y-0.25),
		c,
	)
}

func (io inputOutput) drawBeltIn(mat matrix.Matrix, state DrawState) {
	mat = mat.Mult(io.matrix())
	bounds := rl.NewRectangle(-1, -0.5, 2, 0.5)
	rl.DrawRectangleRec(mat.ApplyRecRec(bounds), state.transformColor(colors.Orange500))
	c := state.transformColor(colors.Black)
	io.drawTri(mat, -0.5, -0.25, c)
	io.drawTri(mat, 0, -0.25, c)
	io.drawTri(mat, 0.5, -0.25, c)
}

func (io inputOutput) drawBeltOut(mat matrix.Matrix, state DrawState) {
	mat = mat.Mult(io.matrix())
	bounds := rl.NewRectangle(-1, 0, 2, 0.5)
	rl.DrawRectangleRec(mat.ApplyRecRec(bounds), state.transformColor(colors.Green500))
	c := state.transformColor(colors.Black)
	io.drawTri(mat, -0.5, 0.25, c)
	io.drawTri(mat, 0, 0.25, c)
	io.drawTri(mat, 0.5, 0.25, c)
}

func (io inputOutput) drawPipeIn(mat matrix.Matrix, state DrawState) {
	mat = mat.Mult(io.matrix())
	bounds := rl.NewRectangle(-0.5, -0.5, 1, 0.5)
	rl.DrawRectangleRec(mat.ApplyRecRec(bounds), state.transformColor(colors.Orange500))
	c := state.transformColor(colors.Black)
	io.drawTri(mat, 0, -0.25, c)
}

func (io inputOutput) drawPipeOut(mat matrix.Matrix, state DrawState) {
	mat = mat.Mult(io.matrix())
	bounds := rl.NewRectangle(-0.5, 0, 1, 0.5)
	rl.DrawRectangleRec(mat.ApplyRecRec(bounds), state.transformColor(colors.Green500))
	c := state.transformColor(colors.Black)
	io.drawTri(mat, 0, 0.25, c)
}

const MAX_INOUT = 6

type inputOutputs struct {
	arr [MAX_INOUT]inputOutput
	len int
}

func (inouts *inputOutputs) UnmarshalJSON(data []byte) error {
	var s []inputOutput
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if len(s) > MAX_INOUT {
		return errors.New("elements count exceeds MAX_INOUT")
	}
	copy(inouts.arr[:], s)
	inouts.len = len(s)
	return err
}

func (inouts inputOutputs) String() string {
	return fmt.Sprintf("%v", inouts.arr[:inouts.len])
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// BuildingDef
////////////////////////////////////////////////////////////////////////////////////////////////////

type BuildingDef struct {
	Class    string
	Category string
	Dims     rl.Vector2
	BeltIn   inputOutputs
	BeltOut  inputOutputs
	PipeIn   inputOutputs
	PipeOut  inputOutputs
}

func (b BuildingDef) String() string {
	s := fmt.Sprintf("{%s(%s) W=%v H=%v", b.Class, b.Category, b.Dims.X, b.Dims.Y)
	if b.BeltIn.len > 0 {
		s += fmt.Sprintf(" BeltIn=%s", b.BeltIn)
	}
	if b.BeltOut.len > 0 {
		s += fmt.Sprintf(" BeltOut=%s", b.BeltOut)
	}
	if b.PipeIn.len > 0 {
		s += fmt.Sprintf(" PipeIn=%s", b.PipeIn)
	}
	if b.PipeOut.len > 0 {
		s += fmt.Sprintf(" PipeOut=%s", b.PipeOut)
	}
	return fmt.Sprintf("%s}", s)
}

type BuildingDefs []BuildingDef

func (defs BuildingDefs) Classes() []string {
	classes := make([]string, len(defs))
	for i, def := range defs {
		classes[i] = def.Class
	}
	return classes
}

func (defs BuildingDefs) Categories() []string {
	var categories []string
	for _, def := range defs {
		if !slices.Contains(categories, def.Category) {
			categories = append(categories, def.Category)
		}
	}
	sort.Strings(categories)
	return categories
}

func (defs BuildingDefs) Index(class string) int {
	for i, def := range defs {
		if def.Class == class {
			return i
		}
	}
	return -1
}
