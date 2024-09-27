package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bonoboris/satisfied/colors"
	"github.com/bonoboris/satisfied/log"
	tfd "github.com/bonoboris/satisfied/tinyfiledialogs"
	"github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// Version of the save file format
	version              = 0
	windowTitle          = "Satisfied"
	extFilter            = "*.satisfied"
	extFilterDesc        = "Satisfied project"
	mandatoryWindowFlags = rl.FlagWindowResizable
	// Default values
	defaultWindowWidth     = 1080
	defaultWindowHeight    = 720
	defaultWindowMaximised = true
	DefaultTargetFPS       = 30
	cacheFile              = "satisfied.cache"
)

var app App

// App contains application global state
type App struct {
	// Application view mode
	Mode AppMode
	// File path
	filepath string
	// draw counts
	drawCounts drawCounts
	// whether the app has panicked
	hasPanicked bool
	// window currentTitle, (used to detect changes)
	currentTitle string
}

type drawCounts struct {
	Buildings int
	Paths     int
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Load / save
////////////////////////////////////////////////////////////////////////////////////////////////////

// resetAll resets the app state to a new project
func (a *App) resetAll() {
	// resets app state
	a.filepath = ""
	a.drawCounts = drawCounts{}
	// clears the scene
	scene.ClearAll()
	// resets most of the app state
	selector.Reset()
	newPath.Reset()
	newBuilding.Reset()
	newTextBox.Reset()
	selection.Reset()
	gui.Reset()
	camera.Reset()
}

// Handles any unsaved changes and returns whether the caller function can continue or not.
func (a *App) checkUnsavedChanges() bool {
	if a.filepath == "" && scene.IsEmpty() || a.filepath != "" && !scene.IsModified() {
		log.Debug("no unsaved changes")
		return true
	}
	switch AskUnsavedChanges() {
	case tfd.ButtonOkYes:
		return a.saveProject(a.filepath)
	case tfd.ButtonNo:
		return true
	case tfd.ButtonCancelNo:
		return false
	default:
		panic("invalid button")
	}
}

// Create a new project and returns whether it was created successfully
func (a *App) newProject() bool {
	log.Info("new project")
	if !a.checkUnsavedChanges() {
		return false
	}
	a.resetAll()
	return true
}

// Open a project file and returns whether it was opened successfully
func (a *App) openProject() bool {
	log.Info("open project")
	if !a.checkUnsavedChanges() {
		return false
	}
	filepath, ok := tfd.OpenFileDialog("Open project", "", []string{extFilter}, extFilterDesc)
	if !ok {
		log.Debug("open project", "action", "cancel")
		return false
	}
	log.Debug("open project", "action", "load", "path", filepath)
	data, err := LoadFromFilepath(filepath)
	if err != nil {
		log.Error("openning project", "action", "cancel", "err", err)
		return false
	}
	a.resetAll()
	a.filepath = filepath
	scene = data.scene
	settings = data.settings
	camera = data.camera
	return true
}

// Save the current project to a new file and returns whether it was saved successfully
func (a *App) saveProjectAs() bool {
	log.Info("save project as")
	filepath := a.filepath
	if filepath == "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filepath = fmt.Sprintf("factory_%s.satisfied", timestamp)
	}
	filepath, ok := tfd.SaveFileDialog("Save project as...", filepath, []string{extFilter}, extFilterDesc)
	if ok && CheckOverwrite(filepath) {
		return a.saveProject(filepath)
	}
	return false
}

// Save the current project to a file and returns whether it was saved successfully
func (a *App) saveProject(filepath string) bool {
	if filepath == "" {
		return a.saveProjectAs()
	}
	if err := SaveToFilepath(filepath); err != nil {
		msg := fmt.Sprintf("Cannot save file: %s\n\nError: %s", filepath, err.Error())
		tfd.MessageBox(
			windowTitle+" - Error saving file",
			RemoveQuotes(msg),
			tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
		return false
	}

	a.filepath = filepath
	scene.ResetModified()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// GUI actions
////////////////////////////////////////////////////////////////////////////////////////////////////

func (a *App) doUndo() Action {
	if a.isNormal() {
		scene.Undo()
	}
	return nil
}

func (a *App) doRedo() Action {
	if a.isNormal() {
		scene.Redo()
	}
	return nil
}

func (a *App) doDelete() Action {
	switch app.Mode {
	case ModeSelection:
		if selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox {
			return selection.doDelete()
		}
	}
	return nil
}

func (a *App) doRotate() Action {
	switch app.Mode {
	case ModeNewPath:
		return newPath.doReverse()
	case ModeNewBuilding:
		return newBuilding.doRotate()
	case ModeSelection:
		return selection.doRotate()
	}
	return nil
}

func (a *App) doDuplicate() Action {
	switch app.Mode {
	case ModeSelection:
		if selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox {
			return selection.doBeginTransformation(SelectionDuplicate, selection.Bounds.Center(), false)
		}
	}
	return nil
}

func (a *App) doDrag() Action {
	switch app.Mode {
	case ModeSelection:
		if selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox {
			return selection.doBeginTransformation(SelectionDrag, selection.Bounds.Center(), false)
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Switch mode
////////////////////////////////////////////////////////////////////////////////////////////////////

// doSwitchMode sets [App.Mode] to the given [AppMode] and reset other modes state
//
// See: [ActionHandler]
func (a *App) doSwitchMode(mode AppMode, resets Resets) Action {
	log.Debug("switchAppMode", "mode", mode, "resets", resets)
	a.Mode = mode
	if resets.Selector {
		selector.Reset()
	}
	if resets.NewPath {
		newPath.Reset()
	}
	if resets.NewBuilding {
		newBuilding.Reset()
	}
	if resets.NewTextBox {
		newTextBox.Reset()
	}
	if resets.Selection {
		selection.Reset()
	}
	if resets.Gui {
		gui.Reset()
	}
	if resets.Camera {
		camera.Reset()
	}

	return nil
}

// Returns wether the app is in [ModeNormal] or [ModeSelection] with [SelectionNormal] sub-mode
func (a *App) isNormal() bool {
	return a.Mode == ModeNormal || a.Mode == ModeSelection && (selection.mode == SelectionNormal || selection.mode == SelectionSingleTextBox)
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Cache functions
////////////////////////////////////////////////////////////////////////////////////////////////////

type cacheData struct {
	LastProject     string
	WindowWidth     int
	WindowHeight    int
	WindowPosX      int
	WindowPosY      int
	WindowMaximized bool
}

func loadCache() (cacheData, error) {
	content, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn("cache file not found", "path", cacheFile)
		} else {
			log.Error("cannot open cache file", "path", cacheFile, "err", err)
		}
		return cacheData{}, err
	}
	var cache cacheData
	err = json.Unmarshal(content, &cache)
	if err != nil {
		log.Error("cannot parse cache file", "path", cacheFile, "err", err)
		return cacheData{}, err
	}
	return cache, nil
}

func saveCache() error {
	cache := cacheData{
		LastProject:     app.filepath,
		WindowWidth:     rl.GetScreenWidth(),
		WindowHeight:    rl.GetScreenHeight(),
		WindowMaximized: rl.IsWindowMaximized(),
	}
	content, err := json.Marshal(cache)
	if err != nil {
		log.Error("cannot format cache data", "path", cacheFile, "err", err)
		return err
	}
	file, err := os.Create(cacheFile)
	if err != nil {
		log.Error("cannot create cache file", "path", cacheFile, "err", err)
		return err
	}
	defer file.Close()
	_, err = file.Write(content)
	if err != nil {
		log.Error("cannot write to cache file", "path", cacheFile, "err", err)
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Main functions
////////////////////////////////////////////////////////////////////////////////////////////////////

type AppOptions struct {
	// Force new project
	New bool
	// A file to load
	File string
	// Target / Max FPS
	Fps int
}

type appParams struct {
	file            string
	fps             int32
	windowHeight    int32
	windowWidth     int32
	windowPosX      int
	windowPosY      int
	windowMaximised bool
}

// Load cache data and merge with passed AppOptions to generate all app initialization parameters
func getAppParams(opts *AppOptions) appParams {
	// defaults
	params := appParams{
		file:            "",
		fps:             DefaultTargetFPS,
		windowHeight:    defaultWindowHeight,
		windowWidth:     defaultWindowWidth,
		windowMaximised: defaultWindowMaximised,
	}
	// override with cache data
	if cache, err := loadCache(); err == nil {
		params.file = cache.LastProject
		params.windowHeight = int32(cache.WindowHeight)
		params.windowWidth = int32(cache.WindowWidth)
		params.windowMaximised = cache.WindowMaximized
		params.windowPosX = cache.WindowPosX
		params.windowPosY = cache.WindowPosY
	}
	if opts != nil {
		// override with options
		if opts.New {
			params.file = ""
		} else if opts.File != "" {
			params.file = opts.File
		}
		if opts.Fps > 0 {
			params.fps = int32(opts.Fps)
		}
	}
	return params
}

func (p appParams) windowFlags() uint32 {
	flags := uint32(mandatoryWindowFlags)
	if p.windowMaximised {
		flags |= rl.FlagWindowMaximized
	}
	return flags
}

// Init initializes the application.
//
// It loads assets, initializes the window, and sets up the default state.
//
// It must only be called once at startup.
func Init(assets embed.FS, opts *AppOptions) error {
	log.Info("initializing application")
	params := getAppParams(opts)
	log.Info("initialization parameters", "params", params)

	// Loading assets
	if err := LoadAssets(assets); err != nil {
		return err
	}
	log.Info("assets loaded")

	// Init window
	rl.SetConfigFlags(rl.FlagWindowHighdpi | rl.FlagMsaa4xHint)
	rl.InitWindow(params.windowWidth, params.windowHeight, windowTitle)
	rl.SetWindowPosition(params.windowPosX, params.windowPosY)
	rl.SetTargetFPS(params.fps)
	rl.SetWindowState(uint32(params.windowFlags()))
	rl.SetExitKey(rl.KeyNull)
	if icon, err := LoadIcon(assets); err == nil {
		rl.SetWindowIcon(*icon)
	}

	log.Info("window initialized")

	// Loading font
	if err := LoadFonts(assets); err != nil {
		return err
	}
	log.Info("fonts loaded")

	// Initializing state
	dims.Update()
	gui.Init()
	camera.Reset()
	log.Info("state initialized")

	app.Mode = ModeNormal

	if params.file != "" {
		if data, err := LoadFromFilepath(params.file); err != nil {
			log.Error("init app with empty scene", "err", err)
		} else {
			app.filepath = params.file
			scene = data.scene
			settings = data.settings
			camera = data.camera
		}
	}
	return nil
}

// Close save cache and cleanup resources used by the application before exiting.
func Close() {
	saveCache()
	rl.UnloadFont(font)
	rl.UnloadFont(labelFont)
	rl.CloseWindow()
}

// ShouldQuit returns true if the application should exit.
func ShouldExit() bool {
	if app.hasPanicked {
		return true
	}

	if rl.WindowShouldClose() {
		return app.checkUnsavedChanges()
	}
	return false
}

// Step updates and draw a frame.
func Step() {
	defer panicHandler()
	update()
	rl.BeginDrawing()
	draw()
	updateAndDrawGui()
	rl.EndDrawing()
}

// getAction dispatches a call to the [GetActionFunc] of the current [AppMode].
//
// Most of the time it will returns `nil`.
func getAction() Action {
	switch app.Mode {
	case ModeNormal:
		return selector.GetAction()
	case ModeNewPath:
		return newPath.GetAction()
	case ModeNewBuilding:
		return newBuilding.GetAction()
	case ModeNewTextBox:
		return newTextBox.GetAction()
	case ModeSelection:
		return selection.GetAction()
	default:
		panic("Invalid app mode")
	}
}

// Dispatch [TargetApp] actions their handler and returns the next [Action] to be performed.
func (app *App) dispatch(action Action) Action {
	switch action := action.(type) {
	case AppActionSwitchMode:
		return app.doSwitchMode(action.Mode, action.Resets)
	default:
		panic(fmt.Sprintf("appDispatch: cannot handle: %T", action))
	}
}

// Dispatch the [Action] to its target [ActionHandler] and returns the next [Action] to be performed.
//
// Most of the time it will returns `nil`.
func dispatchAction(action Action) Action {
	switch action.Target() {
	case TargetApp:
		return app.dispatch(action)
	case TargetGui:
		return gui.Dispatch(action)
	case TargetCamera:
		return camera.Dispatch(action)
	case TargetSelector:
		return selector.Dispatch(action)
	case TargetNewPath:
		return newPath.Dispatch(action)
	case TargetNewBuilding:
		return newBuilding.Dispatch(action)
	case TargetNewTextBox:
		return newTextBox.Dispatch(action)
	case TargetSelection:
		return selection.Dispatch(action)
	default:
		panic("Invalid action target")
	}
}

func (a *App) update() {
	// reset draw counts
	a.drawCounts = drawCounts{}
	// update window title
	if a.filepath != "" {
		title := ""
		if scene.IsModified() {
			title += "•"
		}
		title += fmt.Sprintf("%s - %s", path.Base(a.filepath), windowTitle)
		if a.currentTitle != title {
			log.Debug("app.update", "title", title)
			rl.SetWindowTitle(title)
			a.currentTitle = title
		}
	} else {
		title := fmt.Sprintf("%s - %s", "Unsaved project", windowTitle)
		if a.currentTitle != title {
			log.Debug("app.update", "title", title)
			rl.SetWindowTitle(title)
			a.currentTitle = title
		}
	}
	// check for save shortcut
	if a.isNormal() {
		switch keyboard.Binding() {
		case BindingSaveAs:
			a.saveProjectAs()
		case BindingSave:
			if a.filepath == "" {
				a.saveProjectAs()
			} else {
				a.saveProject(a.filepath)
			}
		}
	}
}

// Update the application state based on mouse and keyboard input(s).
func update() {
	// input updates
	animations.Update()

	keyboard.Update()

	dims.Update()
	camera.Update()
	mouse.Update()
	// FIXME: there some cyclic dependencies between mouse, camera and dims

	app.update()
	scene.Update()

	for action := getAction(); action != nil; action = dispatchAction(action) {
		// empty loop body
		// [GetAction] is called once per frame
		// [Update] is called in a loop until action chain is terminated
		//
		// In most cases, [GetAction] will recursively handle the action chain and return nil.
	}
}

// Draw draws the scene without updating the state.
func draw() {
	rl.ClearBackground(colors.White)
	rl.BeginScissorMode(int32(dims.Scene.X), int32(dims.Scene.Y), int32(dims.Scene.Width), int32(dims.Scene.Height))
	camera.BeginMode2D()

	grid.Draw()

	// draw placed objects
	scene.Draw()

	switch app.Mode {
	case ModeNormal:
		selector.Draw()
	case ModeNewPath:
		newPath.Draw()
	case ModeNewBuilding:
		newBuilding.Draw()
	case ModeNewTextBox:
		newTextBox.Draw()
	case ModeSelection:
		selection.Draw()
	}
	camera.EndMode2D()

	raygui.SetFont(labelFont)
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 24)
	raygui.SetFont(font)
	raygui.SetStyle(raygui.DEFAULT, raygui.TEXT_SIZE, 16)

	rl.EndScissorMode()
}

// UpdateAndDraw combines drawing the GUI, handling GUI inputs and returns an [Action] to be performed.
//
// See: [ActionHandler]
func updateAndDrawGui() {
	for action := gui.UpdateAndDraw(); action != nil; action = dispatchAction(action) {
		// empty loop body
		// [GetAction] is called once per frame
		// [Update] is called in a loop until action chain is terminated
		//
		// In most cases, [Gui.UpdateAndDraw] will recursively handle the action chain and return nil.
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// panic handler
////////////////////////////////////////////////////////////////////////////////////////////////////

const panicTitle = "Satisfied has crashed"

const panicMessageNoBackupFile = `Satisfied has crashed and failed to backup current project.

Error: %s

Failed to backup reason: Cannot find user home directory.

Sorry for the inconvenience, please report this issue to the developer.`

const panicMessageErrBackup = `Satisfied has crashed and failed to backup current project.

Error: %s

Tried to backup in file: %s
But failed because: %s

Sorry for the inconvenience, please report this issue to the developer.`

const panicMessageBackupOk = `Satisfied has crashed.

Error: %s

Current project has been saved in file: %s

Sorry for the inconvenience, please report this issue to the developer.`

func panicHandler() {
	// TODO: save logs, link repo in error message.
	panicErr := recover()
	if panicErr == nil {
		return
	}
	app.hasPanicked = true // schedule app exit
	log.Fatal("application panic", "err", panicErr)

	os.Stderr.Write(fullStack())

	savepath := app.filepath
	if savepath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			msg := fmt.Sprintf(panicMessageNoBackupFile, panicErr)
			tfd.MessageBox(panicTitle, msg, tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
			return
		}
		savepath = NormalizePath(filepath.Join(home, "recover.satisfied"))
	} else {
		savepath = NormalizePath(savepath)
		ext := filepath.Ext(savepath)
		savepath = savepath[:len(savepath)-len(ext)] + ".recover" + ext
	}

	if err := SaveToFilepath(savepath); err != nil {
		msg := fmt.Sprintf(panicMessageErrBackup, panicErr, savepath, err)
		tfd.MessageBox(panicTitle, msg, tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
		return
	}

	msg := fmt.Sprintf(panicMessageBackupOk, panicErr, savepath)
	tfd.MessageBox(panicTitle, msg, tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
}

// fullStack captures the full stack trace, ensuring the buffer is big enough
func fullStack() []byte {
	buf := make([]byte, 4096)
	for {
		buf = make([]byte, len(buf)*2+1024) // Increase buffer size exponentially
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return buf[:n] // If n < len(buf), it means the stack trace is fully captured
		}
	}
}
