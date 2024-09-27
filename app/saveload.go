package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bonoboris/satisfied/log"
	tfd "github.com/bonoboris/satisfied/tinyfiledialogs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// Writing save file
////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	tagVersion               = "#VERSION"
	tagSettingSceneFontSize  = "#SETTING_SCENE_FONT_SIZE"
	tagSettingTickFoundation = "#SETTING_TICK_FOUNDATION"
	tagCameraTarget          = "#CAMERA_TARGET"
	tagCameraZoom            = "#CAMERA_ZOOM"

	textboxClass = "TextBox"
)

// writeTo writes the project state into text format.
//
// All errors originate from the underlying [io.Writer].
func writeTo(w io.Writer) error {
	// // bufSize is kind of low estimation of actual size of the save
	// //   - version line is minimum 10 chars + '\n'
	// //   - the minimum building line is 7 chars + '\n'
	// //   - the minimum path line is 10 chars + '\n'
	// //
	// // Most of the actual lines will be longer as classes are more than 1 char long
	// // and numbers will have multiple digits.
	// bufSize := 10 * (len(s.Paths) + len(s.Buildings) + 1)
	// br := bufio.NewWriterSize(w, bufSize)
	br := bufio.NewWriter(w)
	defer br.Flush()
	// version
	_, err := br.WriteString(fmt.Sprintf("%s=%d\n", tagVersion, version))
	if err != nil {
		return err
	}
	// settings
	_, err = br.WriteString(fmt.Sprintf("%s=%v\n", tagSettingSceneFontSize, settings.FontSize))
	if err != nil {
		return err
	}
	_, err = br.WriteString(fmt.Sprintf("%s=%v\n", tagSettingTickFoundation, settings.TickFoundation))
	if err != nil {
		return err
	}
	// camera
	center := camera.WorldPos(dims.Scene.Center())
	_, err = br.WriteString(fmt.Sprintf("%s=%v %v\n", tagCameraTarget, center.X, center.Y))
	if err != nil {
		return err
	}
	_, err = br.WriteString(fmt.Sprintf("%s=%v\n", tagCameraZoom, camera.Zoom()))
	if err != nil {
		return err
	}

	// buildings
	for _, b := range scene.Buildings {
		class := strings.ReplaceAll(b.Def().Class, " ", "_")
		_, err := br.WriteString(fmt.Sprintf("%s %v %v %d\n", class, b.Pos.X, b.Pos.Y, b.Rot))
		if err != nil {
			return err
		}
	}
	// paths
	for _, p := range scene.Paths {
		class := strings.ReplaceAll(p.Def().Class, " ", "_")
		_, err := br.WriteString(fmt.Sprintf("%s %v %v %v %v\n", class, p.Start.X, p.Start.Y, p.End.X, p.End.Y))
		if err != nil {
			return err
		}
	}
	// textboxes
	for _, tb := range scene.TextBoxes {
		_, err := br.WriteString(fmt.Sprintf("%s %v %v %v %v %v\n",
			textboxClass, tb.Bounds.X, tb.Bounds.Y, tb.Bounds.Width, tb.Bounds.Height,
			strconv.Quote(tb.Content)))
		if err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Reading save file
////////////////////////////////////////////////////////////////////////////////////////////////////

// The subset of the project state that can be saved/loaded
type SaveData struct {
	scene    Scene
	settings Settings
	camera   Camera
}

// ParseWarning represents a warning about a line that can't be parsed and is ignored
type ParseWarning struct {
	// LineNumber is the line number starting from 0
	LineNumber int
	// Line is the line that cannot be parsed
	Line string
	// Message is the warning message
	Message string
	// Err is the error that caused the warning if any
	Err error
}

func (w ParseWarning) String() string {
	if w.Err != nil {
		return fmt.Sprintf("line %d: %s\n\t%s (error: %s)", w.LineNumber, w.Line, w.Message, w.Err.Error())
	}
	return fmt.Sprintf("line %d: %s\n\t%s", w.LineNumber, w.Line, w.Message)
}

func newParseWarning(no int, line string, msg string, err error) ParseWarning {
	return ParseWarning{LineNumber: no, Line: line, Message: msg, Err: err}
}

type DecodeTextError struct {
	Msg     string
	Err     error
	Line    int
	Version int
}

const (
	msgEmpty                = "empty file"
	msgInvalidVersionLine   = "invalid first line, expected '#VERSION=x'"
	msgInvalidVersionNumber = "invalid version, expected a positive integer"
	msgVersionTooHigh       = "version is too high"
	msgInvalidPath          = "invalid path line expected '[class] [startX] [startY] [endX] [endY]'"
	msgInvalidBuilding      = "invalid building line expected '[class] [posX] [posY] [rotation]'"
	msgInvalidTextBox       = "invalid textbox line expected '[class] [posX] [posY] [width] [height] [content]'"
	msgInvalidClass         = "this line is not a tag line nor it starts with a valid class name"
)

func (e DecodeTextError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("line %d: %s (%s)", e.Line, e.Msg, e.Err.Error())
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

func LoadFromText(r io.Reader) (SaveData, []ParseWarning, error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	line := scanner.Text()
	if err := scanner.Err(); err != nil {
		return SaveData{}, nil, err
	}
	if len(line) == 0 {
		return SaveData{}, nil, DecodeTextError{Msg: msgEmpty}
	}
	// parse version
	var ver int
	if _, err := fmt.Sscanf(string(line), tagVersion+"=%d", &ver); err != nil {
		return SaveData{}, nil, DecodeTextError{Msg: msgInvalidVersionLine, Line: 1, Err: err}
	}
	if ver < 0 {
		return SaveData{}, nil, DecodeTextError{Msg: msgInvalidVersionNumber, Line: 1}
	}
	// call version specific function
	switch ver {
	case 0:
		data, warnings := decodeText(scanner)
		return data, warnings, nil
	default:
		return SaveData{}, nil, DecodeTextError{Msg: msgVersionTooHigh, Version: ver, Line: 1}
	}
}

// decodeText decodes the text line by line and returns parsed data and warnings for invalid lines
func decodeText(scanner *bufio.Scanner) (SaveData, []ParseWarning) {
	var data SaveData
	data.camera.Reset() // reset camera to default values
	var warnings []ParseWarning
	no := 2
	var (
		p Path
		b Building
	)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			tag, value, found := strings.Cut(line, "=")
			if !found || tag == "" {
				warnings = append(warnings, newParseWarning(no, line, "invalid tag line format (expected '#TAG=VALUE')", nil))
				continue
			}
			switch tag {
			case tagSettingSceneFontSize:
				f32value, err := ParseFloat32(value)
				if err != nil {
					warnings = append(warnings, newParseWarning(no, line, "cannot parse value as a number", err))
				} else {
					fontSize := FontSize(f32value)
					data.settings.FontSize = max(FontLarge, min(fontSize, FontSmall))
				}
			case tagSettingTickFoundation:
				if value != "true" && value != "false" {
					warnings = append(warnings, newParseWarning(no, line, "expected value to be 'true' or 'false'", nil))
				} else {
					data.settings.TickFoundation = value == "true"
				}
			case tagCameraTarget:
				var target rl.Vector2
				if _, err := fmt.Sscanf(value, "%f %f", &target.X, &target.Y); err != nil {
					warnings = append(warnings, newParseWarning(no, line, "cannot parse value as a pair of numbers", nil))
				} else {
					data.camera.SetTarget(target)
				}
			case tagCameraZoom:
				zoom, err := ParseFloat32(value)
				if err != nil {
					warnings = append(warnings, newParseWarning(no, line, "cannot parse value as a number", err))
				} else {
					data.camera.SetZoom(zoom)
				}
			default:
				warnings = append(warnings, newParseWarning(no, line, "unknown tag line", nil))
			}
			continue
		}

		class, fields, _ := strings.Cut(line, " ")
		class = strings.ReplaceAll(class, "_", " ")
		if class == textboxClass {
			tb, msg, err := parseTextBoxFields(fields)
			if msg != "" {
				warnings = append(warnings, newParseWarning(no, line, msg, err))
			} else {
				data.scene.TextBoxes = append(data.scene.TextBoxes, tb)
			}
		} else if defIdx := pathDefs.Index(string(class)); defIdx >= 0 {
			p.DefIdx = defIdx
			if _, err := fmt.Sscanf(fields, "%f %f %f %f", &p.Start.X, &p.Start.Y, &p.End.X, &p.End.Y); err != nil {
				warnings = append(warnings, newParseWarning(no, line, msgInvalidPath, err))
			} else {
				data.scene.Paths = append(data.scene.Paths, p)
			}
		} else if defIdx := buildingDefs.Index(string(class)); defIdx >= 0 {
			b.DefIdx = defIdx
			if _, err := fmt.Sscanf(fields, "%f %f %d", &b.Pos.X, &b.Pos.Y, &b.Rot); err != nil {
				warnings = append(warnings, newParseWarning(no, line, msgInvalidBuilding, err))
			} else {
				data.scene.Buildings = append(data.scene.Buildings, b)
			}
		} else {
			warnings = append(warnings, newParseWarning(no, line, msgInvalidClass, nil))
		}
		no++
	}

	return data, warnings
}

func parseTextBoxFields(fields string) (TextBox, string, error) {
	var tb TextBox
	var err error
	elts := strings.SplitN(fields, " ", 5)
	if len(elts) != 5 {
		return TextBox{}, msgInvalidTextBox, nil
	}
	tb.Bounds.X, err = ParseFloat32(elts[0])
	if err != nil {
		return TextBox{}, msgInvalidTextBox, err
	}
	tb.Bounds.Y, err = ParseFloat32(elts[1])
	if err != nil {
		return TextBox{}, msgInvalidTextBox, err
	}
	tb.Bounds.Width, err = ParseFloat32(elts[2])
	if err != nil {
		return TextBox{}, msgInvalidTextBox, err
	}
	tb.Bounds.Height, err = ParseFloat32(elts[3])
	if err != nil {
		return TextBox{}, msgInvalidTextBox, err
	}
	tb.Content, err = strconv.Unquote(elts[4])
	if err != nil {
		return TextBox{}, msgInvalidTextBox, err
	}
	return tb, "", nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////////////////////////////////

// SaveToFilepath saves the project state into a file.
func SaveToFilepath(filepath string) error {
	log.Info("saving project", "path", filepath)
	file, err := os.Create(filepath)
	if err != nil {
		log.Error("cannot create file", "path", filepath, "err", err)
		return err
	}
	defer file.Close()
	err = writeTo(file)
	if err != nil {
		log.Error("cannot write to file", "path", filepath, "err", err)
		return err
	}
	log.Info("project saved", "path", filepath)
	return nil
}

// LoadFromFilepath loads the project state from a file.
func LoadFromFilepath(filepath string) (SaveData, error) {
	// On error, log error, display message
	log.Info("loading project", "path", filepath)
	file, err := os.Open(filepath)
	if err != nil {
		log.Error("cannot open file", "path", filepath, "err", err)
		return SaveData{}, err
	}
	defer file.Close()
	data, warnings, err := LoadFromText(file)
	if err != nil {
		log.Error("error parsing save file", "path", filepath, "err", err)
		msg := fmt.Sprintf("Cannot load project: %s\n\nError: %s", filepath, RemoveQuotes(err.Error()))
		tfd.MessageBox(windowTitle+" - Error loading file", msg, tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
	} else {
		if len(warnings) > 0 {
			var sb strings.Builder
			sb.WriteString("Warning(s):\n\n")
			for _, w := range warnings {
				log.Warn("cannot parse line, ignoring", "no", w.LineNumber, "line", w.Line, "msg", w.Message, "err", w.Message)
				sb.WriteString(w.String())
				sb.WriteString("\n")
			}
			sb.WriteString("\nInvalid line(s) will be discarded on next save.")
			tfd.MessageBox(windowTitle+" - Load project", sb.String(), tfd.DialogOk, tfd.IconWarning, tfd.ButtonOkYes)
		} else {
			log.Info("project loaded", "path", filepath)
		}
	}
	return data, err
}

// Checks whether the file exists, if it does asks the user whether to overwrite it;
// returns wheter save
func CheckOverwrite(filepath string) bool {
	_, err := os.Stat(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		log.Error("error checking if file exists", "path", filepath, "err", err)
		msg := fmt.Sprintf("Cannot save project: %s\n\nError: %s", filepath, err.Error())
		tfd.MessageBox(
			windowTitle+" - Error saving file",
			RemoveQuotes(msg),
			tfd.DialogOk, tfd.IconError, tfd.ButtonOkYes)
		return false
	}
	msg := fmt.Sprintf("The file %s already exists.\n\nDo you want to overwrite it ?", filepath)
	choice := tfd.MessageBox(
		windowTitle+" - Save project",
		RemoveQuotes(msg),
		tfd.DialogYesNo, tfd.IconWarning, tfd.ButtonCancelNo)
	return choice == tfd.ButtonOkYes
}

// Ask user what to do with unsaved changes, log and returns user choice
func AskUnsavedChanges() tfd.Button {
	msg := "The current project has unsaved changes.\n\nDo you want to save them ?"
	choice := tfd.MessageBox(
		windowTitle+" - Unsaved project", msg,
		tfd.DialogYesNoCancel, tfd.IconWarning, tfd.ButtonOkYes)

	switch choice {
	case tfd.ButtonOkYes:
		log.Info("unsaved changes", "action", "save")
	case tfd.ButtonNo:
		log.Debug("unsaved changes", "action", "discard")
	case tfd.ButtonCancelNo:
		log.Debug("unsaved changes", "action", "cancel")
	default:
		panic("invalid button")
	}
	return choice
}
