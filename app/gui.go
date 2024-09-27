package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/log"
	"github.com/bonoboris/satisfied/text"
	"github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// Top bar height in px
	TopbarHeight = 50.0
	// Side bar width in px
	SidebarWidth = 300.0
	// Details bar width in px
	DetailsBarWidth = 300.0
	// Status bar height in px
	StatusBarHeight = 30.0
)

var gui = Gui{}

// Gui is a container struct for all the GUI elements
type Gui struct {
	Topbar     guiTopbar
	Sidebar    guiSidebar
	Detailsbar guiDetailsbar
	Statusbar  guiStatusbar
}

// Precompute and store some static data
func (gui *Gui) Init() {
	gui.Sidebar.init()
}

// UpdateAndDraw combines drawing the GUI, handling GUI inputs and returns an [Action] to be performed.
//
// Note: Raygui being an immediate mode GUI, we cannot seperate the update and draw steps.
func (g *Gui) UpdateAndDraw() (action Action) {
	// At most a single non nil action by frame should be returned from updateAndDraw calls
	// we cannot press 2 buttons at the same time
	action = orAction(action, g.Statusbar.updateAndDraw())
	action = orAction(action, g.Detailsbar.updateAndDraw())
	action = orAction(action, g.Sidebar.updateAndDraw())
	action = orAction(action, g.Topbar.updateAndDraw())
	return action
}

// Reset resets GUI state
func (g *Gui) Reset() {
	g.Sidebar.reset()
	g.Detailsbar.reset()
}

// Whether the gui should captures key presses
func (g *Gui) CapturesKeyPress() bool { return g.Detailsbar.textarea.Focused() }

func (g *Gui) traceState() {
	log.Trace("gui.sidebar", "activePath", g.Sidebar.activePath, "activeCategory", g.Sidebar.activeCategory, "activeBuilding", g.Sidebar.activeBuilding)
}

// Dispatch performs an [Gui] action, updating its state, and returns an new action to be performed
//
// See: [ActionHandler]
func (g *Gui) Dispatch(action Action) Action {
	switch action := action.(type) {
	default:
		panic(fmt.Sprintf("Gui.Dispatch: cannot handle: %T", action))
	}
}

type guiTopbar struct{}

func (tb *guiTopbar) updateAndDraw() (action Action) {
	bar := rl.NewRectangle(0, 0, dims.Screen.X, TopbarHeight)

	rl.DrawRectangleRec(bar, colors.Gray100)
	rl.DrawLineV(bar.BottomLeft(), bar.BottomRight(), colors.Gray300)

	// Enable tooltip & set font size to 16
	pTextSize := raygui.GetStyle(raygui.DEFAULT, raygui.TEXT_SIZE)
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 16)
	raygui.EnableTooltip()

	if !app.isNormal() { // begin file controls
		raygui.Disable()
	}
	bounds := rl.NewRectangle(20, 10, 30, 30)
	raygui.SetTooltip("New file")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_FILE_NEW, "")) {
		log.Debug("topbar new file clicked")
		app.newProject()
	}

	bounds.X += 50
	raygui.SetTooltip("Open file")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_FILE_OPEN, "")) {
		log.Debug("topbar open file clicked")
		app.openProject()
	}

	bounds.X += 50
	raygui.SetTooltip("Save file (Ctrl+S)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_FILE_SAVE, "")) {
		log.Debug("topbar save file clicked")
		app.saveProject(app.filepath)
	}

	bounds.X += 50
	raygui.SetTooltip("Save file as...")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_FILE_SAVE_CLASSIC, "")) {
		log.Debug("topbar save file as clicked")
		app.saveProjectAs()
	}
	raygui.Enable() // end file controls

	bounds.X += 50
	rl.DrawLineEx(bounds.TopLeft(), bounds.BottomLeft(), 2, colors.Gray300)

	if !(app.isNormal() && scene.HasUndo()) { // begin undo control
		raygui.Disable()
	}
	bounds.X += 20
	raygui.SetTooltip("Undo (Ctrl+Z)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_UNDO, "")) {
		log.Debug("topbar undo clicked")
		action = app.doUndo()
	}
	raygui.Enable() // end undo control

	if !(app.isNormal() && scene.HasRedo()) { // begin redo control
		raygui.Disable()
	}
	bounds.X += 50
	raygui.SetTooltip("Redo (Ctrl+Y / Ctrl+Shift+Z)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_REDO, "")) {
		log.Debug("topbar redo clicked")
		action = app.doRedo()
	}
	raygui.Enable() // end redo control

	bounds.X += 50
	rl.DrawLineEx(bounds.TopLeft(), bounds.BottomLeft(), 2, colors.Gray300)

	bounds.X += 20
	raygui.SetTooltip("Rotate (R)")
	if !(app.Mode == ModeSelection || app.Mode == ModeNewPath || app.Mode == ModeNewBuilding) { // begin rotate control
		raygui.Disable()
	}
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_ROTATE, "")) {
		log.Debug("topbar rotate clicked")
		action = app.doRotate()
	}
	raygui.Enable() // end rotate control

	if !(app.Mode == ModeSelection && selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox) { // begin selection transform controls
		raygui.Disable()
	}
	bounds.X += 50
	raygui.SetTooltip("Duplicate (D)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_LAYERS, "")) {
		log.Debug("topbar duplicate clicked")
		action = app.doDuplicate()
	}

	bounds.X += 50
	raygui.SetTooltip("Drag (LMB drag / V)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_CURSOR_MOVE_FILL, "")) {
		log.Debug("topbar drag clicked")
		action = app.doDrag()
	}

	bounds.X += 50
	raygui.SetTooltip("Delete (X / Del)")
	if raygui.Button(bounds, raygui.IconText(raygui.ICON_BIN, "")) {
		log.Debug("topbar delete clicked")
		action = app.doDelete()
	}
	raygui.Enable() // end selection transform controls

	// Reset style and tooltip
	raygui.DisableTooltip()
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, pTextSize)

	return action
}

// guiSidebar represents the sidebar of the application
type guiSidebar struct {
	// Active text box index
	activeTextBox int32
	// Active path index
	activePath int32
	// Active category index
	activeCategory int32
	// Active building index (in active category)
	activeBuilding int32

	// Number of paths
	numPath int32
	// Number of categories
	numCategory int32
	// Path toggle group text
	pathText string
	// Category toggle group text
	categoryText string
	// Building toggle group texts
	buildingTexts []string
	// table of indices
	//
	// buildingIndices[activeCategory][activeBuilding]
	// is the index of the active building in buildingDefs
	buildingIndices [][]int
}

const btnHeight = 32

// reset resets sidebar state
//
// We don't reset the active category because it's not what we usually want
func (sb *guiSidebar) reset() {
	sb.activeTextBox = -1
	sb.activePath = -1
	sb.activeBuilding = -1
}

func (sb *guiSidebar) init() {
	paths := pathDefs.Classes()
	sb.numPath = int32(len(paths))
	sb.pathText = strings.Join(paths, ";")

	log.Trace("gui.sidebar", "paths", paths)
	log.Trace("gui.sidebar", "numPath", sb.numPath)
	log.Trace("gui.sidebar", "pathText", sb.pathText)

	categories := buildingDefs.Categories()
	sb.numCategory = int32(len(categories))
	sb.categoryText = strings.Join(categories, "\n")
	sb.buildingIndices = make([][]int, len(categories))
	log.Trace("gui.sidebar", "categories", categories)
	log.Trace("gui.sidebar", "numCategory", sb.numCategory)
	log.Trace("gui.sidebar", "categoryText", sb.categoryText)
	for i, def := range buildingDefs {
		idx := slices.Index(categories, def.Category)
		sb.buildingIndices[idx] = append(sb.buildingIndices[idx], i)
	}
	sb.buildingTexts = make([]string, len(categories))
	for i, cat_idxs := range sb.buildingIndices {
		classes := make([]string, len(cat_idxs))
		for j, idx := range cat_idxs {
			classes[j] = buildingDefs[idx].Class
		}
		sb.buildingTexts[i] = strings.Join(classes, "\n")
		log.Trace("gui.sidebar", "i", i, "buildingIndices[i]", cat_idxs)
		log.Trace("gui.sidebar", "i", i, "buildingTexts[i]", sb.buildingTexts[i])
	}
	sb.activeTextBox = -1
	sb.activePath = -1
	sb.activeCategory = -1
	sb.activeBuilding = -1
	gui.traceState()
}

func (sb *guiSidebar) drawLine(bounds rl.Rectangle, yOffset float32) {
	start := vec2(bounds.X, bounds.Y+yOffset)
	end := vec2(bounds.X+bounds.Width, bounds.Y+yOffset)
	rl.DrawLineEx(start, end, 2, colors.Gray300)
}

func (sb *guiSidebar) drawTextBoxControls(bounds rl.Rectangle, yOffset float32) Action {
	bounds = rl.NewRectangle(bounds.X, bounds.Y+yOffset, bounds.Width, btnHeight)
	newActive := raygui.ToggleGroup(bounds, "Text box", int32(sb.activeTextBox))
	if newActive != sb.activeTextBox {
		// newActive is guaranteed to be != -1 because ToggleGroup returns the index of the newly
		// active toggle (after a click) and we cannot goes from an active one (sb.activeTextBox != 1)
		// to an inactive one (newActive == -1) by clicking on the same toggle
		sb.activeTextBox = newActive
		sb.activePath = -1
		sb.activeCategory = -1
		sb.activeBuilding = -1
		log.Debug("sidebar text box clicked", "index", newActive)
		gui.traceState()
		// newActive matches with actual index in [pathDefs]
		return newTextBox.doInit()
	}
	return nil
}

func (sb *guiSidebar) drawPathsControls(bounds rl.Rectangle, yOffset float32) Action {
	bounds = rl.NewRectangle(bounds.X, bounds.Y+yOffset, bounds.Width/float32(sb.numPath), btnHeight)
	newActive := raygui.ToggleGroup(bounds, sb.pathText, int32(sb.activePath))
	if newActive != sb.activePath {
		// newActive is guaranteed to be != -1 because ToggleGroup returns the index of the newly
		// active toggle (after a click) and we cannot goes from an active one (sb.activePath != 1)
		// to an inactive one (newActive == -1) by clicking on the same toggle
		sb.activePath = newActive
		sb.activeTextBox = -1
		sb.activeCategory = -1
		sb.activeBuilding = -1
		log.Debug("sidebar path clicked", "defIdx", newActive)
		gui.traceState()
		// newActive matches with actual index in [pathDefs]
		return newPath.doInit(int(newActive))
	}
	return nil
}

func (sb *guiSidebar) drawCategoryControls(bounds rl.Rectangle, yOffset float32) Action {
	bounds = rl.NewRectangle(bounds.X, bounds.Y+yOffset, bounds.Width, btnHeight)
	newActive := raygui.ToggleGroup(bounds, sb.categoryText, sb.activeCategory)
	if newActive != sb.activeCategory {
		// newActive is guaranteed to be != -1 because ToggleGroup returns the index of the newly
		// active toggle (after a click) and we cannot goes from an active one (sb.activeCategory != 1)
		// to an inactive one (newActive == -1) by clicking on the same toggle
		sb.activeCategory = newActive
		sb.activeTextBox = -1
		sb.activePath = -1
		sb.activeBuilding = -1
		log.Debug("sidebar category clicked", "catIdx", sb.activeCategory)
		gui.traceState()
		return app.doSwitchMode(ModeNormal, ResetAll().WithGui(false))
	}
	return nil
}

func (sb *guiSidebar) drawBuildingControls(bounds rl.Rectangle, yOffset float32) Action {
	bounds = rl.NewRectangle(bounds.X, bounds.Y+yOffset, bounds.Width, btnHeight)
	newActive := raygui.ToggleGroup(bounds, sb.buildingTexts[sb.activeCategory], sb.activeBuilding)
	if newActive != sb.activeBuilding {
		// newActive is guaranteed to be != -1 because ToggleGroup returns the index of the newly
		// active toggle (after a click) and we cannot goes from an active one (sb.activeBuilding != 1)
		// to an inactive one (newActive == -1) by clicking on the same toggle
		sb.activeBuilding = newActive
		sb.activeTextBox = -1
		sb.activePath = -1
		defIdx := sb.buildingIndices[sb.activeCategory][newActive]
		log.Debug("sidebar building clicked", "defIdx", defIdx)
		gui.traceState()
		return newBuilding.doInit(defIdx)
	}
	return nil
}

func (sb *guiSidebar) updateAndDraw() (action Action) {
	bar := rl.NewRectangle(0, TopbarHeight, SidebarWidth, dims.Screen.Y-TopbarHeight-StatusBarHeight)

	rl.DrawRectangleRec(bar, colors.Gray100)
	rl.DrawLineV(bar.TopRight(), bar.BottomRight(), colors.Gray300)

	pPadding := raygui.GetStyle(raygui.TOGGLE, raygui.GROUP_PADDING)
	pTextSize := raygui.GetStyle(raygui.DEFAULT, raygui.TEXT_SIZE)
	raygui.SetStyle(raygui.TOGGLE, raygui.GROUP_PADDING, 0)
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 24)

	// padded dimensions
	bar = rl.NewRectangle(bar.X+20, bar.Y+20, bar.Width-40, bar.Height-40)

	yOffset := float32(0)
	action = orAction(action, sb.drawTextBoxControls(bar, yOffset))
	yOffset += btnHeight + 20

	action = orAction(action, sb.drawPathsControls(bar, yOffset))
	yOffset += btnHeight + 20

	sb.drawLine(bar, yOffset)
	yOffset += 20

	raygui.SetStyle(raygui.TOGGLE, raygui.GROUP_PADDING, 0)
	action = orAction(action, sb.drawCategoryControls(bar, yOffset))
	yOffset += float32(sb.numCategory)*btnHeight + 20

	if sb.activeCategory > -1 {
		sb.drawLine(bar, yOffset)
		yOffset += 20

		action = orAction(action, sb.drawBuildingControls(bar, yOffset))
	}

	raygui.SetStyle(raygui.TOGGLE, raygui.GROUP_PADDING, pPadding)
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, pTextSize)
	return action
}

type guiDetailsbar struct {
	areaInit bool
	textarea text.Area
}

func textAreaOpts() text.AreaOptions {
	return text.AreaOptions{Font: monoFont, Size: 24, Color: colors.Gray700}
}

func (db *guiDetailsbar) reset() {
	db.areaInit = false
}

func (db *guiDetailsbar) doUpdateTextBoxContent() Action {
	if newText := db.textarea.Text(); newText != db.textBoxContent() {
		tb := scene.TextBoxes[selection.TextBoxIdxs[0]]
		tb.Content = newText
		db.textarea.SetFocused(false)
		scene.ModifyObjects(selection.ObjectSelection, ObjectCollection{TextBoxes: []TextBox{tb}})
	}
	return nil
}

func (db *guiDetailsbar) textBoxContent() string {
	return scene.TextBoxes[selection.TextBoxIdxs[0]].Content
}

func (db *guiDetailsbar) updateAndDraw() Action {
	var action Action
	bar := rl.NewRectangle(
		dims.Screen.X-DetailsBarWidth,
		TopbarHeight,
		SidebarWidth,
		dims.Screen.Y-TopbarHeight-StatusBarHeight)

	rl.DrawRectangleRec(bar, colors.Gray100)
	rl.DrawLineV(bar.TopLeft(), bar.TopRight(), colors.Gray300)

	// padded dimensions
	bar = rl.NewRectangle(bar.X+20, bar.Y+20, bar.Width-40, bar.Height-40)

	// db.textarea.SetBounds(bounds)
	// db.textarea.Draw(keyboard.Pressed)
	if app.Mode == ModeSelection && len(selection.TextBoxIdxs) == 1 && len(selection.BuildingIdxs) == 0 && len(selection.PathIdxs) == 0 {
		titleBounds := bar
		titleBounds.Height = 30
		text.DrawText(titleBounds, "Edit text box content", text.Options{Font: font, Size: 24, Color: colors.Gray700})
		raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 24)

		areaBounds := bar
		areaBounds.Y += 40
		areaBounds.Height = bar.Height - areaBounds.Y - 50

		if !db.areaInit {
			db.textarea = text.NewArea(areaBounds, db.textBoxContent(), textAreaOpts())
			db.areaInit = true
		} else {
			// in case of window resize
			db.textarea.SetBounds(areaBounds)
		}
		db.textarea.SetDisabled(selection.mode != SelectionSingleTextBox)

		// handling KeyEscape here because we want to discard changes on escape
		if keyboard.Pressed == rl.KeyEscape {
			db.textarea.SetText(db.textBoxContent())
		}

		db.textarea.Draw(keyboard.Pressed)

		if keyboard.Pressed == rl.KeyEnter && keyboard.Ctrl {
			action = db.doUpdateTextBoxContent()
		}
		buttonBounds := bar
		buttonBounds.Y = areaBounds.Y + areaBounds.Height + 10
		buttonBounds.Height = 30
		buttonBounds.Width /= 2
		if selection.mode != SelectionSingleTextBox {
			raygui.Disable()
		}
		if raygui.Button(buttonBounds, "Reset (Esc)") {
			db.textarea.SetText(db.textBoxContent())
		}
		buttonBounds.X += buttonBounds.Width
		if raygui.Button(buttonBounds, "Update (Ctrl+Enter)") {
			action = db.doUpdateTextBoxContent()
		}

		raygui.Enable()
	} else {
		db.reset()
	}
	return action
}

type guiStatusbar struct{}

func (sb *guiStatusbar) updateAndDraw() Action {
	bar := rl.NewRectangle(0, dims.Screen.Y-StatusBarHeight, dims.Screen.X, StatusBarHeight)
	rl.DrawRectangleRec(bar, colors.Gray100)
	rl.DrawLineV(bar.TopLeft(), bar.TopRight(), colors.Gray300)

	// left aligned text
	lpos := bar.TopLeft().Add(vec2(5, 5))
	ltext := fmt.Sprintf("FPS=% 3d | %12v | Building Draws=%d | Path Draws=%d", int(rl.GetFPS()), app.Mode, app.drawCounts.Buildings, app.drawCounts.Paths)
	rl.DrawTextEx(font, ltext, lpos, 24, 1, colors.Gray700)

	// right aligned text
	rtext := fmt.Sprintf("X:%5d  Y:%5d", int(mouse.SnappedPos.X), int(mouse.SnappedPos.Y))
	width := rl.MeasureTextEx(font, rtext, 24, 1).X
	rpos := bar.TopRight().Add(vec2(-5-width, 5))
	rl.DrawTextEx(font, rtext, rpos, 24, 1, colors.Gray700)

	return nil
}

// orAction returns the first non nil action, or nil if both are nil
func orAction(a, b Action) Action {
	if a != nil {
		return a
	}
	return b
}
