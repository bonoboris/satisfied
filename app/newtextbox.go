package app

import (
	"fmt"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/log"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var newTextBox NewTextBox

type NewTextBox struct {
	firstCornerPlaced bool
	pt1, pt2          rl.Vector2
}

func (ntb NewTextBox) traceState(key, val string) {
	if key != "" && val != "" {
		log.Trace("newTextBox", key, val, "pt1", ntb.pt1, "pt2", ntb.pt2, "firstCornerPlaced", ntb.firstCornerPlaced)
	} else {
		log.Trace("newTextBox", "pt1", ntb.pt1, "pt2", ntb.pt2, "firstCornerPlaced", ntb.firstCornerPlaced)
	}
}

// Reset resets the [NewTextBox] state
func (ntb *NewTextBox) Reset() {
	ntb.traceState("before", "Reset")
	log.Debug("newTextBox.reset")
	ntb.firstCornerPlaced = false
	ntb.pt1 = rl.Vector2{}
	ntb.pt2 = rl.Vector2{}
	ntb.traceState("after", "Reset")
}

// GetAction processes inputs in [ModeNewTextBox], and returns an action to be performed.
//
// See: [GetActionFunc]
func (ntb *NewTextBox) GetAction() (action Action) {
	app.Mode.Assert(ModeNewTextBox)

	switch keyboard.Binding() {
	case BindingEscape:
		if ntb.firstCornerPlaced {
			return ntb.doInit()
		} else {
			return app.doSwitchMode(ModeNormal, ResetAll())
		}
	}

	if mouse.InScene && mouse.Left.Released {
		if !ntb.firstCornerPlaced {
			return ntb.doPlaceStart()
		} else {
			return ntb.doPlace()
		}
	}
	if mouse.InScene && !mouse.Left.Down {
		return ntb.doMoveTo(mouse.SnappedPos)
	}
	return nil
}

func (ntb *NewTextBox) doInit() Action {
	ntb.traceState("before", "doInit")
	log.Debug("newTextBox.doInit")
	ntb.pt1 = rl.Vector2{}
	ntb.pt2 = rl.Vector2{}
	ntb.firstCornerPlaced = false
	ntb.traceState("after", "doInit")
	return app.doSwitchMode(ModeNewTextBox, ResetAll().WithNewTextBox(false))
}

func (ntb *NewTextBox) doMoveTo(pos rl.Vector2) Action {
	ntb.traceState("before", "doMoveTo")
	log.Trace("newTextBox.doMoveTo", "pos", pos) // moving by mouse -> tracing
	app.Mode.Assert(ModeNewTextBox)
	if !ntb.firstCornerPlaced {
		ntb.pt1 = pos
		ntb.pt2 = pos
	} else {
		ntb.pt2 = pos
	}
	ntb.traceState("after", "doMoveTo")
	return nil
}

func (ntb *NewTextBox) doPlaceStart() Action {
	ntb.traceState("before", "doPlaceStart")
	log.Debug("newTextBox.doPlaceStart")
	ntb.firstCornerPlaced = true
	ntb.traceState("after", "doPlaceStart")
	return nil
}

func (ntb *NewTextBox) doPlace() Action {
	ntb.traceState("before", "doPlace")
	log.Debug("newTextBox.doPlace")
	app.Mode.Assert(ModeNewTextBox)
	assert(ntb.firstCornerPlaced, "text box start not placed")
	tb := TextBox{Bounds: rl.NewRectangleCorners(ntb.pt1, ntb.pt2), Content: textBoxDefaultText}
	scene.AddTextBox(tb)
	idx := len(scene.TextBoxes) - 1
	return selection.doInitSelection(ObjectSelection{TextBoxIdxs: []int{idx}})
}

// Dispatch performs an [NewTextBox] action, updating its state, and returns an new action to be performed
//
// See: [ActionHandler]
func (np *NewTextBox) Dispatch(action Action) Action {
	switch action := action.(type) {
	case NewTextBoxActionInit:
		return np.doInit()
	case NewTextBoxActionMoveTo:
		return np.doMoveTo(action.Pos)
	case NewTextBoxActionPlaceStart:
		return np.doPlaceStart()
	case NewTextBoxActionPlace:
		return np.doPlace()

	default:
		panic(fmt.Sprintf("NewTextBox.Dispatch: cannot handle: %T", action))
	}
}

func (np NewTextBox) Draw() {
	if !np.firstCornerPlaced {
		rl.DrawRectangleV(np.pt1, vec2(textBoxMinSize, textBoxMinSize), colors.Gray300)
	} else {
		rl.DrawRectangleRec(rl.NewRectangleCorners(np.pt1, np.pt2), colors.Gray300)
	}
}
