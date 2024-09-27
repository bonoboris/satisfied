package app

import (
	"github.com/bonoboris/satisfied/log"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Scene holds the scene objects (buildings and paths)
var scene Scene

// Scene holds the scene objects (buildings and paths)
type Scene struct {
	ObjectCollection
	// History of scene operations (undo / redo)
	history []sceneOp
	// Current history position:
	//   - history[:historyPos] all have been done
	//   - history[historyPos:] all have been undone (if existing)
	historyPos int

	// History position the last time the scene was saved
	savedHistoryPos int

	// The scene object currently hovered by the mouse
	Hovered Object
	// was in modified state last frame
	wasModified bool
}

func (s Scene) traceState(key, val string) {
	if log.WillTrace() {
		if key != "" && val != "" {
			log.Trace("scene", key, val)
		}

		for i, b := range s.Buildings {
			log.Trace("scene.buildings", "i", i, "value", b)
		}
		for i, p := range s.Paths {
			log.Trace("scene.paths", "i", i, "value", p)
		}
		for i, tb := range s.TextBoxes {
			log.Trace("scene.textboxes", "i", i, "value", tb)
		}
		log.Trace("scene", "wasModified", s.wasModified, "historyPos", s.historyPos, "savedHistoryPos", s.savedHistoryPos)
		for i, op := range s.history {
			log.Trace("scene.history", "i", i, "op", op)
		}
		log.Trace("scene", "hovered", s.Hovered)
	}
}

// ClearAll clears the scene of all objects, history, ...
func (s *Scene) ClearAll() {
	log.Debug("scene.clearAll")
	s.Buildings = s.Buildings[:0]
	s.Paths = s.Paths[:0]
	s.TextBoxes = s.TextBoxes[:0]
	s.history = s.history[:0]
	s.historyPos = 0
	s.savedHistoryPos = 0
	s.wasModified = false
	s.Hovered = Object{}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// sceneOp (undo / redo)
////////////////////////////////////////////////////////////////////////////////////////////////////

type sceneOpType string

const (
	SceneOpAdd    sceneOpType = "add"
	SceneOpDelete sceneOpType = "delete"
	SceneOpModify sceneOpType = "modify"
)

// sceneOp represents a scene operation
type sceneOp struct {
	// Type is the type of the operation
	Type sceneOpType
	// Sel is the selection the operation acts on (empty for [SceneOpAdd])
	Sel ObjectSelection
	// Old is the objects before the operation (empty for [SceneOpAdd])
	//
	// - in [SceneOpDelete] Old.Paths contains only the deleted paths ([ObjectSelection.FullPathIdxs])
	// - in [SceneOpModify] Old.Paths contains all the paths ([ObjectSelection.AnyPathIdxs])
	Old ObjectCollection
	// New is the objects after the operation (empty for [SceneOpDelete])
	New ObjectCollection
}

func (op sceneOp) traceState() {
	switch op.Type {
	case SceneOpAdd:
		log.Trace("scene.operation", "type", "add", "New", op.New)
	case SceneOpDelete:
		log.Trace("scene.operation", "type", "delete", "Sel", op.Sel, "Old", op.Old)
	case SceneOpModify:
		log.Trace("scene.operation", "type", "modify", "Sel", op.Sel, "Old", op.Old, "New", op.New)
	default:
		panic("invalid scene operation type")
	}
}

// do performs the operation
func (op sceneOp) do(s *Scene) {
	s.traceState("before", "sceneOp.do")
	op.traceState()
	log.Info("scene.operation", "do", string(op.Type))
	switch op.Type {

	case SceneOpAdd:
		log.Debug("scene.operation.add", "action", "do",
			"num_paths", len(op.New.Paths),
			"num_buildings", len(op.New.Buildings),
			"num_textboxes", len(op.New.TextBoxes))

		s.Paths = append(s.Paths, op.New.Paths...)
		s.Buildings = append(s.Buildings, op.New.Buildings...)
		s.TextBoxes = append(s.TextBoxes, op.New.TextBoxes...)

	case SceneOpDelete:
		pathIdxs := op.Sel.FullPathIdxs()
		log.Debug("scene.operation.delete", "action", "do",
			"paths", pathIdxs, "buildings", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		s.Paths = SwapDeleteMany(s.Paths, op.Sel.FullPathIdxs())
		s.Buildings = SwapDeleteMany(s.Buildings, op.Sel.BuildingIdxs)
		s.TextBoxes = SwapDeleteMany(s.TextBoxes, op.Sel.TextBoxIdxs)

	case SceneOpModify:
		pathIdxs := op.Sel.AnyPathIdxs()
		log.Debug("scene.operation.modify", "action", "do",
			"paths", pathIdxs, "buildings", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		for i, idx := range pathIdxs {
			s.Paths[idx] = op.New.Paths[i]
		}
		for i, idx := range op.Sel.BuildingIdxs {
			s.Buildings[idx] = op.New.Buildings[i]
		}
		for i, idx := range op.Sel.TextBoxIdxs {
			s.TextBoxes[idx] = op.New.TextBoxes[i]
		}

	default:
		panic("invalid scene operation type")
	}
	s.traceState("after", "sceneOp.do")
}

// redo performs the operation and returns the new selection, if any
func (op sceneOp) redo(s *Scene) ObjectSelection {
	s.traceState("before", "sceneOp.redo")
	op.traceState()
	log.Info("scene.operation", "redo", string(op.Type))

	var newSel ObjectSelection

	switch op.Type {

	case SceneOpAdd:
		log.Debug("scene.operation.add", "action", "redo",
			"num_paths", len(op.New.Paths),
			"num_buildings", len(op.New.Buildings),
			"num_textboxes", len(op.New.TextBoxes))
		s.Paths = append(s.Paths, op.New.Paths...)
		s.Buildings = append(s.Buildings, op.New.Buildings...)
		s.TextBoxes = append(s.TextBoxes, op.New.TextBoxes...)

		newSel = ObjectSelection{
			BuildingIdxs: Range(len(s.Buildings)-len(op.New.Buildings), len(s.Buildings)),
			TextBoxIdxs:  Range(len(s.TextBoxes)-len(op.New.TextBoxes), len(s.TextBoxes)),
		}
		n := len(s.Paths)
		for i := range len(op.New.Paths) {
			newSel.PathIdxs = append(newSel.PathIdxs, PathSel{Idx: n + i, Start: true, End: true})
		}
		newSel.recomputeBounds(s.ObjectCollection)

	case SceneOpDelete:
		pathIdxs := op.Sel.FullPathIdxs()
		log.Debug("scene.operation.delete", "action", "redo",
			"paths", pathIdxs, "buildings", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		s.Paths = SwapDeleteMany(s.Paths, pathIdxs)
		s.Buildings = SwapDeleteMany(s.Buildings, op.Sel.BuildingIdxs)
		s.TextBoxes = SwapDeleteMany(s.TextBoxes, op.Sel.TextBoxIdxs)

	case SceneOpModify:
		pathIdxs := op.Sel.AnyPathIdxs()
		log.Debug("scene.operation.modify", "action", "redo",
			"paths", pathIdxs, "buildings", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		for i, idx := range pathIdxs {
			s.Paths[idx] = op.New.Paths[i]
		}
		for i, idx := range op.Sel.BuildingIdxs {
			s.Buildings[idx] = op.New.Buildings[i]
		}
		for i, idx := range op.Sel.TextBoxIdxs {
			s.TextBoxes[idx] = op.New.TextBoxes[i]
		}
		newSel = op.Sel
		newSel.recomputeBounds(s.ObjectCollection)

	default:
		panic("invalid scene operation type")
	}
	s.traceState("after", "sceneOp.redo")
	return newSel
}

// undo performs the operation and returns the new selection, if any
func (op sceneOp) undo(s *Scene) ObjectSelection {
	s.traceState("before", "sceneOp.undo")
	op.traceState()
	log.Info("scene.operation", "undo", string(op.Type))

	var newSel ObjectSelection

	switch op.Type {
	case SceneOpAdd:
		log.Debug("scene.operation.add", "action", "undo",
			"num_paths", len(op.New.Paths),
			"num_buildings", len(op.New.Buildings),
			"num_textboxes", len(op.New.TextBoxes))
		s.Paths = s.Paths[:len(s.Paths)-len(op.New.Paths)]
		s.Buildings = s.Buildings[:len(s.Buildings)-len(op.New.Buildings)]
		s.TextBoxes = s.TextBoxes[:len(s.TextBoxes)-len(op.New.TextBoxes)]

	case SceneOpDelete:
		pathIdxs := op.Sel.FullPathIdxs()
		log.Debug("scene.operation.delete", "action", "undo",
			"paths", pathIdxs, "buildinds", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		s.Paths = SwapInsertMany(s.Paths, pathIdxs, op.Old.Paths)
		s.Buildings = SwapInsertMany(s.Buildings, op.Sel.BuildingIdxs, op.Old.Buildings)
		s.TextBoxes = SwapInsertMany(s.TextBoxes, op.Sel.TextBoxIdxs, op.Old.TextBoxes)

		newSel = op.Sel

	case SceneOpModify:
		pathIdxs := op.Sel.AnyPathIdxs()
		log.Debug("scene.operation.modify", "action", "redo",
			"paths", pathIdxs, "buildings", op.Sel.BuildingIdxs, "textboxes", op.Sel.TextBoxIdxs)
		for i, idx := range pathIdxs {
			s.Paths[idx] = op.Old.Paths[i]
		}
		for i, idx := range op.Sel.BuildingIdxs {
			s.Buildings[idx] = op.Old.Buildings[i]
		}
		for i, idx := range op.Sel.TextBoxIdxs {
			s.TextBoxes[idx] = op.Old.TextBoxes[i]
		}

		newSel = op.Sel
		newSel.recomputeBounds(s.ObjectCollection)

	default:
		panic("invalid scene operation type")
	}
	s.traceState("after", "sceneOp.undo")
	return newSel
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Scene Modifiers methods
////////////////////////////////////////////////////////////////////////////////////////////////////

// doSceneOp adds the given operation to the scene history and performs it
func (s *Scene) doSceneOp(op sceneOp) {
	s.history = s.history[:s.historyPos] // trim any undone operations
	op.do(s)                             // actually perform the operation
	s.history = append(s.history, op)    // append the operation to the history
	s.historyPos++                       // increment history position
	s.Hovered = Object{}                 // invalidate hovered object just in case
}

// AddPath adds the given path to the scene.
//
// No validity check is performed.
func (s *Scene) AddPath(path Path) {
	s.doSceneOp(sceneOp{Type: SceneOpAdd, New: ObjectCollection{Paths: []Path{path}}})
}

// AddBuilding adds the given building to the scene.
//
// No validity check is performed.
func (s *Scene) AddBuilding(building Building) {
	s.doSceneOp(sceneOp{Type: SceneOpAdd, New: ObjectCollection{Buildings: []Building{building}}})
}

// AddTextBox adds the given text box to the scene.
func (s *Scene) AddTextBox(tb TextBox) {
	s.doSceneOp(sceneOp{Type: SceneOpAdd, New: ObjectCollection{TextBoxes: []TextBox{tb}}})
}

// AddObjects adds the given paths and buildings to the scene.
//
// No validity checks is performed.
func (s *Scene) AddObjects(col ObjectCollection) {
	s.doSceneOp(sceneOp{Type: SceneOpAdd, New: col.clone()})
}

// DeleteObjects deletes the given paths and buildings from the scene.
func (s *Scene) DeleteObjects(sel ObjectSelection) {
	sel = sel.clone()
	op := sceneOp{Type: SceneOpDelete, Sel: sel}
	op.Old.Buildings = CopyIdxs(op.Old.Buildings, s.Buildings, sel.BuildingIdxs)
	op.Old.Paths = CopyIdxs(op.Old.Paths, s.Paths, sel.FullPathIdxs())
	op.Old.TextBoxes = CopyIdxs(op.Old.TextBoxes, s.TextBoxes, sel.TextBoxIdxs)
	s.doSceneOp(op)
}

// ModifyObjects updates the given paths and buildings in the scene.
//
// No validity checks is performed.
func (s *Scene) ModifyObjects(sel ObjectSelection, new ObjectCollection) {
	sel = sel.clone()
	op := sceneOp{Type: SceneOpModify, Sel: sel, New: new.clone()}
	op.Old.Buildings = CopyIdxs(op.Old.Buildings, s.Buildings, sel.BuildingIdxs)
	op.Old.Paths = CopyIdxs(op.Old.Paths, s.Paths, sel.AnyPathIdxs())
	op.Old.TextBoxes = CopyIdxs(op.Old.TextBoxes, s.TextBoxes, sel.TextBoxIdxs)
	s.doSceneOp(op)
}

// Undo tries to undo the last operation, and returns whether it has, and the action to be performed.
func (s *Scene) Undo() (bool, Action) {
	if s.historyPos > 0 {
		s.historyPos-- // decrement history position
		op := s.history[s.historyPos]
		s.Hovered = Object{} // invalidate hovered object just in case
		// will switch to [ModeSelection] or [ModeNormal] if new selection is empty
		return true, selection.doInitSelection(op.undo(s))
	}
	log.Warn("cannot undo operation", "reason", "no more operations to undo")
	return false, nil
}

// Redo tries to redo the last undone operation, and returns whether it has, and the action to be performed.
func (s *Scene) Redo() (bool, Action) {
	if s.historyPos < len(s.history) {
		op := s.history[s.historyPos]
		s.historyPos++       // increment history position
		s.Hovered = Object{} // invalidate hovered object just in case
		// will switch to [ModeSelection] or [ModeNormal] if new selection is empty
		return true, selection.doInitSelection(op.redo(s))
	}
	log.Warn("cannot redo operation", "reason", "no more operations to redo")
	return false, nil
}

// HasUndo returns true if there are more undo operations to perform
func (s *Scene) HasUndo() bool { return s.historyPos > 0 }

// HasRedo returns true if there are more redo operations to perform
func (s *Scene) HasRedo() bool { return s.historyPos < len(s.history) }

// IsModified returns true if the scene has been modified since last save
func (s *Scene) IsModified() bool {
	return s.historyPos != s.savedHistoryPos
}

// ResetModified resets the scene modified flag
func (s *Scene) ResetModified() {
	s.traceState("before", "ResetModified")
	s.savedHistoryPos = s.historyPos
	log.Debug("scene.resetModified", "savedHistoryPos", s.savedHistoryPos)
	s.traceState("after", "ResetModified")
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Scene other methods
////////////////////////////////////////////////////////////////////////////////////////////////////

// GetObjectAt returns the object at the given position (world coordinates)
//
// If multiple objects are at the position returns first one on this list:
//   - selected path with highest index: start / end over body
//   - selected building with highest index
//   - selected text box with highest index
//   - normal path with highest index: start / end over body
//   - normal building with highest index
//   - normal text box with highest index
//
// This is (mostly) the reverse of [Scene.Draw] order to make viewing/selecting masked objects easier.
//
// If no object is found, returns an zero-valued [Object]
func (s Scene) GetObjectAt(pos rl.Vector2) Object {
	for i := len(selection.PathIdxs) - 1; i >= 0; i-- {
		elt := selection.PathIdxs[i]
		p := s.Paths[elt.Idx]
		if elt.Start && p.CheckStartCollisionPoint(pos) {
			return Object{Type: TypePathStart, Idx: elt.Idx}
		}
		if elt.End && p.CheckEndCollisionPoint(pos) {
			return Object{Type: TypePathEnd, Idx: elt.Idx}
		}
		if p.CheckCollisionPoint(pos) {
			return Object{Type: TypePath, Idx: elt.Idx}
		}
	}

	for i := len(selection.BuildingIdxs) - 1; i >= 0; i-- {
		if s.Buildings[selection.BuildingIdxs[i]].Bounds().CheckCollisionPoint(pos) {
			return Object{Type: TypeBuilding, Idx: selection.BuildingIdxs[i]}
		}
	}
	for i := len(selection.TextBoxIdxs) - 1; i >= 0; i-- {
		if s.TextBoxes[selection.TextBoxIdxs[i]].Bounds.CheckCollisionPoint(pos) {
			return Object{Type: TypeTextBox, Idx: selection.TextBoxIdxs[i]}
		}
	}

	// TODO: do not check selected paths / buildings again ?
	for i := len(s.Paths) - 1; i >= 0; i-- {
		p := s.Paths[i]
		if p.CheckStartCollisionPoint(pos) {
			return Object{Type: TypePathStart, Idx: i}
		}
		if p.CheckEndCollisionPoint(pos) {
			return Object{Type: TypePathEnd, Idx: i}
		}
		if p.CheckCollisionPoint(pos) {
			return Object{Type: TypePath, Idx: i}
		}
	}

	for i := len(s.Buildings) - 1; i >= 0; i-- {
		if s.Buildings[i].Bounds().CheckCollisionPoint(pos) {
			return Object{Type: TypeBuilding, Idx: i}
		}
	}

	for i := len(s.TextBoxes) - 1; i >= 0; i-- {
		if s.TextBoxes[i].Bounds.CheckCollisionPoint(pos) {
			return Object{Type: TypeTextBox, Idx: i}
		}
	}

	return Object{}
}

// Update hovered object
func (s *Scene) Update() (action Action) {
	s.Hovered = s.GetObjectAt(mouse.Pos)

	if app.isNormal() && keyboard.Ctrl {
		switch keyboard.Binding() {
		case BindingUndo:
			_, action = s.Undo()
		case BindingRedo:
			_, action = s.Redo()
		}
	}
	return action
}

func (s Scene) IsBuildingValid(building Building, ignore int) bool {
	bounds := building.Bounds()
	for i, b := range s.Buildings {
		if i == ignore {
			continue
		}
		if b.Bounds().CheckCollisionRec(bounds) {
			return false
		}
	}
	return true
}

func (s Scene) IsPathValid(path Path) bool {
	return !path.Start.Equals(path.End)
}

// draws the scene objects accounting for selection / selector
func (s Scene) drawWithSel() {
	var state DrawState
	var buildingIt MaskIterator
	var pathIt PathSelMaskIterator
	var textBoxIt MaskIterator
	if app.Mode == ModeSelection {
		buildingIt = selection.BuildingsIterator()
		pathIt = selection.PathsIterator()
		textBoxIt = selection.TextBoxesIterator()
		switch selection.mode {
		case SelectionNormal, SelectionSingleTextBox:
			state = DrawSkip
		case SelectionDrag, SelectionTextBoxResize:
			state = DrawShadow
		case SelectionDuplicate:
			state = DrawClicked
		}
	} else {
		buildingIt = selector.BuildingsIterator()
		pathIt = selector.PathsIterator()
		textBoxIt = selector.TextBoxesIterator()
		state = DrawSkip
	}

	if app.Mode == ModeSelection && selection.mode == SelectionDrag {
		// in drag mode, draw the whole path as shadow
		for _, p := range s.Paths {
			start, end := pathIt.Next()
			if start || end {
				p.Draw(state)
			} else {
				p.Draw(DrawNormal)
			}
		}
	} else {
		for _, b := range s.Paths {
			start, end := pathIt.Next()
			if start {
				b.DrawStart(state)
			} else {
				b.DrawStart(DrawNormal)
			}
			if end {
				b.DrawEnd(state)
			} else {
				b.DrawEnd(DrawNormal)
			}
			if start && end {
				b.DrawBody(state)
			} else {
				b.DrawBody(DrawNormal)
			}
		}
	}
	for _, b := range s.Buildings {
		if buildingIt.Next() {
			b.Draw(state)
		} else {
			b.Draw(DrawNormal)
		}
	}
	for _, b := range s.TextBoxes {
		if textBoxIt.Next() {
			b.Draw(state, false)
		} else {
			b.Draw(DrawNormal, false)
		}
	}
}

// draws selection / selector objects that have been skipped in [Scene.drawWithSel]
func (s Scene) drawSelSkipped() {
	var sel ObjectSelection
	var state DrawState
	if app.Mode == ModeSelection {
		sel = selection.ObjectSelection
		state = DrawSelected
	} else {
		sel = selector.ObjectSelection
		state = DrawHovered
	}

	for _, idx := range sel.BuildingIdxs {
		s.Buildings[idx].Draw(state)
	}

	for _, elt := range sel.PathIdxs {
		p := s.Paths[elt.Idx]
		if elt.Start && elt.End {
			p.DrawBody(state)
		}
		if elt.Start {
			p.DrawStart(state)
		}
		if elt.End {
			p.DrawEnd(state)
		}
	}

	for _, idx := range sel.TextBoxIdxs {
		s.TextBoxes[idx].Draw(state, selection.mode == SelectionSingleTextBox)
	}
}

// Draw scene objects
func (s Scene) Draw() {
	if app.Mode == ModeSelection || app.Mode == ModeNormal && selector.selecting {
		s.drawWithSel()
	} else {
		for _, b := range s.Paths {
			b.Draw(DrawNormal)
		}
		for _, b := range s.Buildings {
			b.Draw(DrawNormal)
		}
		for _, b := range s.TextBoxes {
			b.Draw(DrawNormal, false)
		}
	}

	// draw selection on top
	if app.Mode == ModeSelection && (selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox) ||
		app.Mode == ModeNormal && selector.selecting {
		s.drawSelSkipped()
	}

	// draw hovered object
	if !s.Hovered.IsEmpty() {
		if app.Mode == ModeNormal && !selector.selecting {
			s.Hovered.Draw(DrawNormal | DrawHovered)
		} else if app.Mode == ModeSelection && (selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox) {
			if selection.Contains(s.Hovered) {
				s.Hovered.Draw(DrawSelected | DrawHovered)
			} else {
				s.Hovered.Draw(DrawNormal | DrawHovered)
			}
		}
	}
}
