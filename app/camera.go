// camera - camera state

package app

import (
	"fmt"

	"github.com/bonoboris/satisfied/log"
	"github.com/bonoboris/satisfied/math32"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// Zoom default level (px/wu) wu=world unit
	zoomDefault = 10.
	// Factor to zoom by
	zoomFactor = 1.25
	// Min zoom level = zoomDefault * zoomFactor^-5 (~ x0.40 default)
	zoomMin = zoomDefault * 256. / 625.
	// Max zoom level = zoomDefault * zoomFactor^5 (~ x2.5 default)
	zoomMax = zoomDefault * 625. / 256.
	// Ammount to move the camera by on arrow key press
	moveDelta = 100.
	// Ammount to zoom by on middle mouse button drag
	zoomPerPx = 1.0 / 100.
)

var camera = Camera{}

// Holds camera state
type Camera struct {
	// inner state
	camera rl.Camera2D
	// true if camera is currently zooming (middle mouse button down)
	Zooming bool
	// zoom at position
	ZoomAt rl.Vector2
}

func (c Camera) traceState(key, val string) {
	if key != "" && val != "" {
		log.Trace("camera", key, val, "zoom", c.camera.Zoom, "target", c.camera.Target, "offset", c.camera.Offset, "zooming", c.Zooming, "zoomAt", c.ZoomAt)
	} else {
		log.Trace("camera", "zoom", c.camera.Zoom, "target", c.camera.Target, "offset", c.camera.Offset, "zooming", c.Zooming, "zoomAt", c.ZoomAt)
	}
}

func (c *Camera) Reset() {
	c.camera.Zoom = zoomDefault
	c.camera.Target = vec2(0, 0)
	c.camera.Offset = dims.Scene.Center()
	c.Zooming = false
	c.ZoomAt = rl.Vector2{}
}

// Zoom returns the current zoom level
func (c *Camera) Zoom() float32 { return c.camera.Zoom }

// WorldPos converts screen position to world position
func (c *Camera) WorldPos(screenPos rl.Vector2) rl.Vector2 {
	return rl.GetScreenToWorld2D(screenPos, c.camera)
}

// ScreenPos converts world position to screen position
func (c *Camera) ScreenPos(worldPos rl.Vector2) rl.Vector2 {
	return rl.GetWorldToScreen2D(worldPos, c.camera)
}

// BeginMode2D enters raylib 2D mode
func (c *Camera) BeginMode2D() { rl.BeginMode2D(c.camera) }

// EndMode2D exits raylib 2D mode
func (c *Camera) EndMode2D() { rl.EndMode2D() }

// Update processes inputs, and directly execute the [TargetCamera] action to be performed.
//
// [TargetCamera] actions does not have follow up actions.
func (c *Camera) Update() {
	// arrow keys
	if app.Mode == ModeNormal {
		switch keyboard.Binding() {
		case BindingRight:
			c.doPan(vec2(-moveDelta, 0))
		case BindingLeft:
			c.doPan(vec2(+moveDelta, 0))
		case BindingDown:
			c.doPan(vec2(0, -moveDelta))
		case BindingUp:
			c.doPan(vec2(0, +moveDelta))
		}
	}

	// +, -, =
	switch keyboard.Binding() {
	case BindingZoomOut:
		c.doZoom(-1, dims.Scene.Center())
	case BindingZoomIn:
		c.doZoom(+1, dims.Scene.Center())
	case BindingZoomReset:
		c.doReset()
	}

	// mouse inputs
	if mouse.InScene {
		if mouse.Right.Down {
			// paning
			c.doPan(mouse.ScreenDelta)
		} else if mouse.Middle.Pressed {
			c.Zooming = true
			c.ZoomAt = mouse.ScreenPos
		} else if mouse.Middle.Released {
			c.Zooming = false
		} else if mouse.Middle.Down {
			// zooming by middle mouse button drag
			if math32.Abs(mouse.ScreenDelta.X) > math32.Abs(mouse.ScreenDelta.Y) {
				c.doZoom(mouse.ScreenDelta.X*zoomPerPx, c.ZoomAt)
			} else {
				c.doZoom(mouse.ScreenDelta.Y*zoomPerPx, c.ZoomAt)
			}
		} else if mouse.Wheel != 0 {
			// zooming by mouse wheel
			c.doZoom(mouse.Wheel, mouse.ScreenPos)
		}
	}
}

// doReset resets camera state (default zoom, target (0,0) and offset middle of the scene)
func (c *Camera) doReset() Action {
	c.traceState("before", "doReset")
	log.Debug("camera.doReset")
	c.Reset()
	c.traceState("after", "doReset")
	return nil
}

// doZoom zooms the camera by a given amount at a given position
func (c *Camera) doZoom(by float32, at rl.Vector2) Action {
	c.traceState("before", "doZoom")
	if mouse.Middle.Down {
		log.Trace("camera.doZoom", "by", by, "at", at) // zooming by mouse wheel -> tracing
	} else {
		log.Debug("camera.doZoom", "by", by, "at", at) // zooming by keyboard -> tracing
	}
	// Set target at world position
	c.camera.Target = c.WorldPos(at)
	// Set offset at screen position
	c.camera.Offset = at
	// Change zoom
	newZoom := c.camera.Zoom * math32.Pow(zoomFactor, by)
	c.camera.Zoom = min(max(newZoom, zoomMin), zoomMax)
	c.traceState("after", "doZoom")
	return nil
}

// doPan pans the camera by a given amount
func (c *Camera) doPan(by rl.Vector2) Action {
	c.traceState("before", "doPan")
	if mouse.Right.Down { // panning by mouse movement -> tracing
		log.Trace("camera.doPan", "by", by)
	} else {
		log.Debug("camera.doPan", "by", by) // panning by keyboard -> tracing
	}
	// Set target at world position
	c.camera.Offset = c.camera.Offset.Add(by)
	c.traceState("after", "doPan")
	return nil
}

// Dispatch performs a [Camera] action, updating its state, and returns an new action to be performed
//
// Note: All camera actions returns nil (no follow up)
//
// See: [ActionHandler]
func (c *Camera) Dispatch(action Action) Action {
	switch action := action.(type) {
	case CameraActionReset:
		return c.doReset()
	case CameraActionZoom:
		return c.doZoom(action.By, action.At)
	case CameraActionPan:
		return c.doPan(action.By)

	default:
		panic(fmt.Sprintf("Camera.Dispatch: cannot handle: %T", action))
	}
}

// Set sets the camera target; offset is set to the center of the scene area
func (c *Camera) SetTarget(target rl.Vector2) {
	c.camera.Target = target
	c.camera.Offset = dims.Scene.Center()
}

// SetZoom sets the camera zoom
func (c *Camera) SetZoom(zoom float32) {
	c.camera.Zoom = zoom
}
