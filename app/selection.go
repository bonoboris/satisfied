// selection - Handle selection

package app

import (
	"fmt"
	"slices"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/log"
	"github.com/bonoboris/satisfied/matrix"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var selection Selection

////////////////////////////////////////////////////////////////////////////////////////////////////
// Selection
////////////////////////////////////////////////////////////////////////////////////////////////////

type Selection struct {
	ObjectSelection
	// Selection mode
	mode SelectionMode
	// transformed selection data (in drag or duplicate mode)
	transform selectionTransform
	// update transform position on mouse down
	transformMoveOnMouseDown bool
}

func (s Selection) traceState(key, val string) {
	if log.WillTrace() {
		if key != "" && val != "" {
			log.Trace("selection", key, val, "mode", s.mode)
		}
		log.Trace("selection", "buildingIdxs", s.BuildingIdxs)
		log.Trace("selection", "pathIdxs", s.PathIdxs)
		log.Trace("selection", "textboxIdxs", s.TextBoxIdxs)
		log.Trace("selection", "mode", s.mode, "bounds", s.Bounds)
		s.transform.traceState()
	}
}

func (s *Selection) Reset() {
	log.Debug("selection.reset")
	s.BuildingIdxs = s.BuildingIdxs[:0]
	s.PathIdxs = s.PathIdxs[:0]
	s.TextBoxIdxs = s.TextBoxIdxs[:0]
	s.Bounds = rl.NewRectangle(0, 0, 0, 0)
	s.mode = SelectionNormal
	s.transform.reset()
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// SelectionMode
////////////////////////////////////////////////////////////////////////////////////////////////////

// Represents a selection sub mode
type SelectionMode int

const (
	// Normal selection
	SelectionNormal SelectionMode = iota
	// A single text box is selected, allow text editing in details panel & text box resizing
	SelectionSingleTextBox
	// Selection is dragged
	SelectionDrag
	// Selection is being duplicated
	SelectionDuplicate
	// A single text box is being resized
	SelectionTextBoxResize
)

func (m SelectionMode) String() string {
	switch m {
	case SelectionNormal:
		return "SelectionNormal"
	case SelectionSingleTextBox:
		return "SelectionSingleTextBox"
	case SelectionDrag:
		return "SelectionDrag"
	case SelectionDuplicate:
		return "SelectionDuplicate"
	case SelectionTextBoxResize:
		return "SelectionTextBoxResize"
	default:
		return "Invalid"
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// selection transform
////////////////////////////////////////////////////////////////////////////////////////////////////

// Represent a transformation of the selection (drag or duplicate)
type selectionTransform struct {
	// Transformation state

	// transformation rotation
	rot int32
	// start position of the transformation
	startPos rl.Vector2
	// end position of the transformation
	endPos rl.Vector2

	// Transformation results / data
	ObjectCollection
	// invalid transformed paths mask
	invalidPaths []bool
	// invalid transformed buildings mask
	invalidBuildings []bool
	// whether the every transformed object is valid
	isValid bool
	// bounds of the transformed selection
	bounds rl.Rectangle

	// transformed building bounds buffer (reduce allocs)
	_buildingBounds []rl.Rectangle
}

func (st selectionTransform) traceState() {
	if log.WillTrace() {
		for i, p := range st.Paths {
			log.Trace("selectionTransform.paths", "i", i, "value", p, "invalid", st.invalidPaths[i])
		}
		for i, b := range st.Buildings {
			log.Trace("selectionTransform.buildings", "i", i, "value", b, "invalid", st.invalidBuildings[i])
		}
		for i, tb := range st.TextBoxes {
			log.Trace("selectionTransform.textboxes", "i", i, "value", tb)
		}
		log.Trace("selectionTransform", "isValid", st.isValid, "bounds", st.bounds)
		log.Trace("selectionTransform", "rot", st.rot, "startPos", st.startPos, "endPos", st.endPos)
	}
}

func (st *selectionTransform) reset() {
	st.rot = 0
	st.startPos = rl.Vector2{}
	st.endPos = rl.Vector2{}

	st.Paths = st.Paths[:0]
	st.Buildings = st.Buildings[:0]
	st.TextBoxes = st.TextBoxes[:0]
	st.invalidPaths = st.invalidPaths[:0]
	st.invalidBuildings = st.invalidBuildings[:0]
	st.isValid = false
	st.bounds = rl.Rectangle{}
}

func (s selectionTransform) isIdentity() bool {
	translate := grid.Snap(s.endPos.Subtract(s.startPos))
	return s.rot%360 == 0 && translate.X == 0 && translate.Y == 0
}

// Matrix returns the rotation matrix of the selection
func (s selectionTransform) transformMatrix(baseBounds rl.Rectangle) matrix.Matrix {
	center := baseBounds.Center()
	translate := grid.Snap(s.endPos.Subtract(s.startPos))
	return matrix.NewTranslateV(translate.Add(center)).Rotate(s.rot).TranslateV(center.Negate())
}

// recompute recomputes the transformed objects and whether they are valid
func (st *selectionTransform) recompute(sel ObjectSelection, mode SelectionMode) {
	if mode == SelectionTextBoxResize {
		// endPos is the new bottom right corner
		st.isValid = true
		st.TextBoxes = st.TextBoxes[:0]
		tb := scene.TextBoxes[sel.TextBoxIdxs[0]]
		tb.Bounds = rl.NewRectangleCorners(tb.Bounds.TopLeft(), grid.Snap(st.endPos))
		st.TextBoxes = append(st.TextBoxes, tb)
		return
	}

	// fast path for identity transform
	// TODO: not copying anything would be faster
	if st.isIdentity() {
		switch mode {
		case SelectionDuplicate:
			pathIdxs := sel.FullPathIdxs()
			st.Buildings = CopyIdxs(st.Buildings, scene.Buildings, sel.BuildingIdxs)
			st.invalidBuildings = Repeat(st.invalidBuildings, true, len(sel.BuildingIdxs))
			st.Paths = CopyIdxs(st.Paths, scene.Paths, pathIdxs)
			st.invalidPaths = Repeat(st.invalidPaths, true, len(pathIdxs))
			st.TextBoxes = CopyIdxs(st.TextBoxes, scene.TextBoxes, sel.TextBoxIdxs)
			st.isValid = false
			st.bounds = sel.Bounds
		default:
			pathIdxs := sel.AnyPathIdxs()
			st.Buildings = CopyIdxs(st.Buildings, scene.Buildings, sel.BuildingIdxs)
			st.invalidBuildings = Repeat(st.invalidBuildings, false, len(sel.BuildingIdxs))
			st.Paths = CopyIdxs(st.Paths, scene.Paths, pathIdxs)
			st.invalidPaths = Repeat(st.invalidPaths, false, len(pathIdxs))
			st.TextBoxes = CopyIdxs(st.TextBoxes, scene.TextBoxes, sel.TextBoxIdxs)
			st.isValid = true
			st.bounds = sel.Bounds
		}
		return
	}

	// TODO: store transformMatrix and recompute only when needed

	st.isValid = true

	var pathIdxs []int
	if mode == SelectionDuplicate {
		// we only want to duplicate paths that are entirely inside the selection
		pathIdxs = sel.FullPathIdxs()
	} else {
		pathIdxs = sel.AnyPathIdxs()
	}

	ntb := len(sel.TextBoxIdxs)
	nb := len(sel.BuildingIdxs)
	np := len(pathIdxs)

	// clears slices
	st._buildingBounds = slices.Grow(st._buildingBounds[:0], nb)
	st.Buildings = slices.Grow(st.Buildings[:0], nb)
	st.invalidBuildings = slices.Grow(st.invalidBuildings[:0], nb)
	st.Paths = slices.Grow(st.Paths[:0], np)
	st.invalidPaths = slices.Grow(st.invalidPaths[:0], np)
	st.TextBoxes = slices.Grow(st.TextBoxes[:0], ntb)

	mat := st.transformMatrix(sel.Bounds)
	st.bounds = mat.ApplyRecRec(sel.Bounds)

	// Buildings
	for _, idx := range sel.BuildingIdxs {
		b := scene.Buildings[idx]
		b.Pos = mat.ApplyV(b.Pos)
		b.Rot = (b.Rot + st.rot) % 360
		st.Buildings = append(st.Buildings, b)
	}

	// TextBoxes
	for _, idx := range sel.TextBoxIdxs {
		tb := scene.TextBoxes[idx]
		pos := mat.ApplyV(tb.Bounds.Position())
		tb.Bounds.X = pos.X
		tb.Bounds.Y = pos.Y
		st.TextBoxes = append(st.TextBoxes, tb)
	}

	// Paths & invalidPaths
	switch mode {
	case SelectionDuplicate:
		for _, idx := range pathIdxs {
			p := scene.Paths[idx]
			p.Start = mat.ApplyV(p.Start)
			p.End = mat.ApplyV(p.End)
			st.Paths = append(st.Paths, p)
			if p.IsValid() {
				st.invalidPaths = append(st.invalidPaths, false)
			} else {
				st.invalidPaths = append(st.invalidPaths, true)
				st.isValid = false
			}
		}
	case SelectionDrag, SelectionNormal:
		for _, elt := range sel.PathIdxs {
			p := scene.Paths[elt.Idx]
			if elt.Start {
				p.Start = mat.ApplyV(p.Start)
			}
			if elt.End {
				p.End = mat.ApplyV(p.End)
			}
			st.Paths = append(st.Paths, p)
			if p.IsValid() {
				st.invalidPaths = append(st.invalidPaths, false)
			} else {
				st.invalidPaths = append(st.invalidPaths, true)
				st.isValid = false
			}
		}
	}

	// precompute transformed building bounds & initialize invalid to false
	for i := range nb {
		st._buildingBounds = append(st._buildingBounds, st.Buildings[i].Bounds())
		st.invalidBuildings = append(st.invalidBuildings, false)
	}

	isSelectedIt := NewMaskIterator(sel.BuildingIdxs)
	for _, sb := range scene.Buildings {
		sb := sb.Bounds()
		// TODO: use st.Buildings bounds only in the skip condition ?
		if mode != SelectionDuplicate && isSelectedIt.Next() || !st.bounds.CheckCollisionRec(sb) {
			// skip:
			//   - scene building in selection (except when duplicating)
			//   - scene building outside transformation outer bounds
			continue
		} else {
			// check against every transformed building
			for i, bounds := range st._buildingBounds {
				// no need to call CheckCollisionRec if st.invalidBuildings[i] is already true
				if !st.invalidBuildings[i] && bounds.CheckCollisionRec(sb) {
					st.isValid = false
					st.invalidBuildings[i] = true
				}
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Selection methods
////////////////////////////////////////////////////////////////////////////////////////////////////

// GetAction processes inputs in [ModeSelection] and returns an action to be performed.
//
// See: [GetActionFunc]
func (s *Selection) GetAction() Action {
	app.Mode.Assert(ModeSelection)
	switch s.mode {
	case SelectionNormal, SelectionSingleTextBox:
		switch keyboard.Binding() {
		case BindingEscape:
			return app.doSwitchMode(ModeNormal, ResetAll())
		case BindingDuplicate:
			// Duplicate use center of current selection as start position
			return s.doBeginTransformation(SelectionDuplicate, s.Bounds.Center(), false)
		case BindingDrag:
			return s.doBeginTransformation(SelectionDrag, s.Bounds.Center(), false)
		case BindingDelete:
			return s.doDelete()
		case BindingRotate:
			return s.doRotate()

		case BindingLeft:
			return s.doMoveBy(vec2(-1, 0))
		case BindingRight:
			return s.doMoveBy(vec2(+1, 0))
		case BindingUp:
			return s.doMoveBy(vec2(0, -1))
		case BindingDown:
			return s.doMoveBy(vec2(0, +1))
		}
		if mouse.Left.Pressed && mouse.InScene {
			switch {
			case scene.Hovered.IsEmpty():
				return selector.doInit(mouse.Pos)
			case selection.Contains(scene.Hovered):
				if s.mode == SelectionSingleTextBox && scene.TextBoxes[s.TextBoxIdxs[0]].HandleRect().CheckCollisionPoint(mouse.Pos) {
					return s.doBeginTransformation(SelectionTextBoxResize, mouse.Pos, true)
				}
				// Drag use mouse position as start position
				return s.doBeginTransformation(SelectionDrag, mouse.Pos, true)
			default:
				return s.doInitSingleDrag(scene.Hovered, mouse.Pos)
			}
		}
	case SelectionDuplicate, SelectionDrag, SelectionTextBoxResize:
		// TODO: Implement arrow keys nudging ?
		switch keyboard.Binding() {
		case BindingEscape:
			return s.doEndTransformation(true)
		case BindingRotate:
			return s.doRotate()
		}
		switch {
		case mouse.Left.Released:
			return s.doEndTransformation(false)
		case mouse.InScene && (s.transformMoveOnMouseDown || !mouse.Left.Down):
			return s.doMoveTo(mouse.Pos)
		}
	default:
		panic("invalid selection mode")
	}
	return nil
}

// reset [Selection.mode]
//
// - empty selection -> [ModeNormal]
// - single text box -> [ModeSelection] in [SelectionSingleTextBox]
// - other -> [ModeSelection] in [SelectionNormal]
func (s *Selection) resetMode() (AppMode, Resets) {
	if len(s.BuildingIdxs) == 0 && len(s.PathIdxs) == 0 {
		if len(s.TextBoxIdxs) == 0 {
			log.Debug("selection.resetMode", "appMode", ModeNormal)
			return ModeNormal, ResetAll()
		} else if len(s.TextBoxIdxs) == 1 {
			log.Debug("selection.resetMode", "appMode", ModeSelection, "mode", SelectionSingleTextBox)
			s.mode = SelectionSingleTextBox
			return ModeSelection, ResetAll().WithSelection(false)
		}
	}
	log.Debug("selection.resetMode", "appMode", ModeSelection, "mode", SelectionNormal)
	s.mode = SelectionNormal
	return ModeSelection, ResetAll().WithSelection(false)
}

// Initializes a new selection from a single object in [SelectionDrag] mode
func (s *Selection) doInitSingleDrag(obj Object, pos rl.Vector2) Action {
	log.Debug("selection.doInitSingleDrag", "obj", obj, "pos", pos)
	s.Reset()

	switch obj.Type {
	case TypeBuilding:
		s.BuildingIdxs = append(s.BuildingIdxs, obj.Idx)
		s.Bounds = scene.Buildings[obj.Idx].Bounds()
	case TypePath:
		s.PathIdxs = append(s.PathIdxs, PathSel{Idx: obj.Idx, Start: true, End: true})
		s.Bounds = rl.NewRectangleCorners(scene.Paths[obj.Idx].Start, scene.Paths[obj.Idx].End)
	case TypePathStart:
		s.PathIdxs = append(s.PathIdxs, PathSel{Idx: obj.Idx, Start: true})
		s.Bounds = rl.NewRectangleV(scene.Paths[obj.Idx].Start, rl.Vector2{})
	case TypePathEnd:
		s.PathIdxs = append(s.PathIdxs, PathSel{Idx: obj.Idx, End: true})
		s.Bounds = rl.NewRectangleV(scene.Paths[obj.Idx].End, rl.Vector2{})
	case TypeTextBox:
		s.TextBoxIdxs = append(s.TextBoxIdxs, obj.Idx)
		s.Bounds = scene.TextBoxes[obj.Idx].Bounds
	default:
		panic("invalid object type")
	}
	s.mode = SelectionDrag
	s.transformMoveOnMouseDown = true
	s.transform.startPos = pos
	s.transform.endPos = pos
	s.transform.recompute(s.ObjectSelection, SelectionDrag) // noop transformation ->uses fast path

	s.traceState("after", "doInitSingle")
	return app.doSwitchMode(ModeSelection, ResetAll().WithSelection(false))
}

// doInitSelection initializes a new selection from an [ObjectSelection]
//
// - empty selection -> [ModeNormal]
// - single text box -> [ModeSelection] in [SelectionSingleTextBox]
// - other -> [ModeSelection] in [SelectionNormal]
func (s *Selection) doInitSelection(sel ObjectSelection) Action {
	log.Debug("selection.doInitSelection", "selected", sel)
	sel.copy(&s.ObjectSelection)
	s.transform.reset()
	s.transform.startPos = s.Bounds.Center()
	s.transform.endPos = s.Bounds.Center()
	appMode, resets := s.resetMode()
	s.traceState("after", "doInitSelection")
	return app.doSwitchMode(appMode, resets)
}

func (s *Selection) doDelete() Action {
	s.traceState("before", "doDelete")
	log.Debug("selection.doDelete")
	app.Mode.Assert(ModeSelection)

	assert(s.mode == SelectionNormal || s.mode == SelectionSingleTextBox, "cannot delete selection in "+s.mode.String())

	scene.DeleteObjects(s.ObjectSelection)

	s.traceState("after", "doDelete")
	return app.doSwitchMode(ModeNormal, ResetAll())
}

func (s *Selection) doBeginTransformation(mode SelectionMode, pos rl.Vector2, moveOnMouseDown bool) Action {
	s.traceState("before", "doBeginTransformation")
	log.Debug("selection.doBeginTransformation", "mode", mode, "pos", pos)
	app.Mode.Assert(ModeSelection)

	assert(mode == SelectionDrag || mode == SelectionDuplicate || mode == SelectionTextBoxResize, "invalid selection transform mode")

	s.transformMoveOnMouseDown = moveOnMouseDown

	// special cases for only path start/end selected for duplicate
	if mode == SelectionDuplicate && len(s.BuildingIdxs) == 0 && len(s.TextBoxIdxs) == 0 {
		noFullPath := true
		for _, elt := range s.PathIdxs {
			if elt.Start && elt.End {
				noFullPath = false
				break
			}
		}
		if noFullPath {
			if len(s.PathIdxs) == 1 {
				// only one path start/end selected switch to newpath mode
				log.Debug("selection.doBeginTransformation", "action", "newpath", "mode", mode, "reason", "single path ending selected")
				return newPath.doInit(scene.Paths[s.PathIdxs[0].Idx].DefIdx)
			} else {
				// TODO: would be nice to have a multi newpath mode
				// for now do nothing
				log.Debug("selection.doBeginTransformation", "action", "skiped", "mode", mode, "reason", "only path endings selected")
				s.traceState("after", "doBeginTransformation")
				return nil
			}
		}
	}

	s.transform.reset()

	s.mode = mode
	s.transform.startPos = pos
	s.transform.endPos = pos
	s.transform.recompute(s.ObjectSelection, mode) // noop transformation -> uses fast path

	s.traceState("after", "doBeginTransformation")
	return nil
}

func (s *Selection) doMoveBy(delta rl.Vector2) Action {
	s.traceState("before", "doMoveBy")
	log.Debug("selection.doMoveBy", "delta", delta, "selection.mode", s.mode)
	app.Mode.Assert(ModeSelection)

	switch s.mode {
	case SelectionNormal, SelectionSingleTextBox:
		// instantly move by
		center := s.Bounds.Center()
		s.transform.startPos = center
		s.transform.endPos = center.Add(delta)
		s.transform.rot = 0
		s.transform.recompute(s.ObjectSelection, s.mode)
		s.traceState("after", "doMoveBy")
		return s.doEndTransformation(false)
	default:
		s.transform.endPos = s.transform.endPos.Add(delta)
		s.transform.recompute(s.ObjectSelection, s.mode)
		s.traceState("after", "doMoveBy")
		return nil
	}
}

func (s *Selection) doMoveTo(pos rl.Vector2) Action {
	s.traceState("before", "doMoveTo")
	log.Trace("selection.doMoveTo", "pos", pos) // moving by mouse -> tracing
	app.Mode.Assert(ModeSelection)

	switch s.mode {
	case SelectionNormal, SelectionSingleTextBox:
		// instantly move to
		center := s.Bounds.Center()
		s.transform.startPos = center
		s.transform.endPos = pos
		s.transform.rot = 0
		s.transform.recompute(s.ObjectSelection, s.mode)
		s.traceState("after", "doMoveTo")
		return s.doEndTransformation(false)
	default:
		s.transform.endPos = pos
		s.transform.recompute(s.ObjectSelection, s.mode)
		s.traceState("after", "doMoveTo")
		return nil
	}
}

func (s *Selection) doRotate() Action {
	s.traceState("before", "doRotate")
	log.Debug("selection.doRotate", "selection.mode", s.mode)
	app.Mode.Assert(ModeSelection)

	switch s.mode {
	case SelectionSingleTextBox, SelectionTextBoxResize:
		// no-op text boxes cannot be rotated
		s.traceState("after", "doRotate")
		return nil
	case SelectionNormal:
		// instantly rotate
		center := s.Bounds.Center()
		s.transform.startPos = center
		s.transform.endPos = center
		s.transform.rot = 90
		s.transform.recompute(s.ObjectSelection, s.mode)
		s.traceState("after", "doRotate")
		return s.doEndTransformation(false)
	default:
		s.transform.rot += 90
		s.transform.recompute(s.ObjectSelection, s.mode)

		s.traceState("after", "doRotate")
		return nil
	}
}

func (s *Selection) doEndTransformation(discard bool) Action {
	s.traceState("before", "doEndTransformation")
	log.Debug("selection.doEndTransformation", "discard", discard, "selection.mode", s.mode)
	app.Mode.Assert(ModeSelection)

	if !discard && s.transform.isValid && !s.transform.isIdentity() {
		switch s.mode {
		case SelectionDuplicate:
			scene.AddObjects(s.transform.ObjectCollection)
		default:
			scene.ModifyObjects(s.ObjectSelection, s.transform.ObjectCollection)
			s.Bounds = s.transform.bounds
		}
	}

	if discard || s.mode != SelectionDuplicate {
		s.transform.reset()
		appMode, resets := s.resetMode()
		s.recomputeBounds(scene.ObjectCollection)
		s.traceState("after", "doEndTransformation")
		return app.doSwitchMode(appMode, resets)
	}
	s.traceState("after", "doEndTransformation")
	return nil
}

// Dispatch performs a [Selection] action, updating its state, and returns an new action to be performed
//
// See: [ActionHandler]
func (s *Selection) Dispatch(action Action) Action {
	switch action := action.(type) {
	case SelectionActionInitSingleDrag:
		return s.doInitSingleDrag(action.Object, action.Pos)
	case SelectionActionDelete:
		return s.doDelete()
	case SelectionActionBeginTransformation:
		return s.doBeginTransformation(action.Mode, action.Pos, action.MoveOnMouseDown)
	case SelectionActionMoveTo:
		return s.doMoveTo(action.Pos)
	case SelectionActionRotate:
		return s.doRotate()
	case SelectionActionEndTransformation:
		return s.doEndTransformation(action.Discard)

	default:
		panic(fmt.Sprintf("Selection.Dispatch: cannot handle: %T", action))
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Draw
////////////////////////////////////////////////////////////////////////////////////////////////////

const boundsThickness = 3.

func drawSelectionBounds(bounds rl.Rectangle, isValid bool) {
	px := 1 / camera.Zoom()
	dt := boundsThickness * px
	bounds = rl.NewRectangleV(bounds.TopLeft().SubtractValue(dt), bounds.Size().AddValue(2*dt))
	if isValid {
		rl.DrawRectangleLinesEx(bounds, dt, colors.WithAlpha(colors.Blue500, 0.5))
	} else {
		rl.DrawRectangleLinesEx(bounds, dt, colors.WithAlpha(colors.Red500, 0.5))
		rl.DrawRectangleRec(bounds, colors.WithAlpha(colors.Red500, 0.15))
	}
}

// draws the transformed selection
func (s *selectionTransform) draw(state DrawState) {
	drawSelectionBounds(s.bounds, s.isValid)
	for i, p := range s.Paths {
		if s.invalidPaths[i] {
			p.Draw(DrawInvalid)
		} else {
			p.Draw(state)
		}
	}
	for i, b := range s.Buildings {
		if s.invalidBuildings[i] {
			b.Draw(DrawInvalid)
		} else {
			b.Draw(state)
		}
	}
	for _, tb := range s.TextBoxes {
		tb.Draw(state, selection.mode == SelectionTextBoxResize)
	}
}

// Draws the selection (and the transformed selection if any)
func (s Selection) Draw() {
	switch s.mode {
	case SelectionNormal, SelectionSingleTextBox:
		// only draw the selection rectangle, buildings and paths are drawn in [Scene.Draw]
		drawSelectionBounds(s.Bounds, true)
	case SelectionDrag, SelectionTextBoxResize:
		s.transform.draw(DrawClicked)
	case SelectionDuplicate:
		s.transform.draw(DrawNew)
	}
}
