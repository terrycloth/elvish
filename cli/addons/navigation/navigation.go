// Package navigation provides the functionality of navigating the filesystem.
package navigation

import (
	"sort"
	"strings"
	"unicode"

	"github.com/elves/elvish/cli"
	"github.com/elves/elvish/cli/el"
	"github.com/elves/elvish/cli/el/colview"
	"github.com/elves/elvish/cli/el/layout"
	"github.com/elves/elvish/cli/el/listbox"
	"github.com/elves/elvish/cli/el/textview"
	"github.com/elves/elvish/styled"
)

// Config contains the configuration needed for the navigation functionality.
type Config struct {
	// Key binding.
	Binding el.Handler
	// Underlying filesystem.
	Cursor Cursor
}

// Start starts the navigation function.
func Start(app *cli.App, cfg Config) {
	cursor := cfg.Cursor
	if cursor == nil {
		cursor = NewOSCursor()
	}
	w := colview.Widget{
		OverlayHandler: cfg.Binding,
		Weights:        func(n int) []int { return []int{1, 3, 4} },
	}
	w.OnLeft = func() {
		// Remember the name of the current directory before ascending.
		currentName := ""
		current, err := cursor.Current()
		if err == nil {
			currentName = current.Name()
		}

		err = cursor.Ascend()
		if err != nil {
			app.Notify(err.Error())
		} else {
			updateState(&w, cursor, currentName)
		}
	}
	w.OnRight = func() {
		currentCol, ok := w.CopyColViewState().Columns[1].(*listbox.Widget)
		if !ok {
			return
		}
		state := currentCol.CopyListboxState()
		if state.Items.Len() == 0 {
			return
		}
		selected := state.Items.(fileItems)[state.Selected]
		if !selected.Mode().IsDir() {
			// Check if the file is a symlink to a directory.
			mode, err := selected.DeepMode()
			if err != nil {
				app.Notify(err.Error())
				return
			}
			if !mode.IsDir() {
				return
			}
		}
		err := cursor.Descend(selected.Name())
		if err != nil {
			app.Notify(err.Error())
		} else {
			updateState(&w, cursor, "")
		}
	}
	updateState(&w, cursor, "")
	app.MutateAppState(func(s *cli.State) { s.Listing = &w })
	app.Redraw(false)
}

func updateState(w *colview.Widget, cursor Cursor, selectName string) {
	var parentCol, currentCol, previewCol el.Widget

	parent, err := cursor.Parent()
	if err == nil {
		parentCol = makeWidget(parent)
	} else {
		parentCol = makeErrWidget(err)
	}

	current, err := cursor.Current()
	if err == nil {
		currentCol = makeWidget(current)
		tryToSelectName(parentCol, current.Name())
		if selectName != "" {
			tryToSelectName(currentCol, selectName)
		}
	} else {
		currentCol = makeErrWidget(err)
		tryToSelectNothing(parentCol)
	}

	previewCol = layout.Empty{}
	if list, ok := currentCol.(*listbox.Widget); ok {
		// Build the preview column.
		state := list.CopyListboxState()
		if state.Items.Len() > 0 {
			previewCol = makeWidget(state.Items.(fileItems)[state.Selected])
		}
		// Update the preview column whenever the selection changes.
		list.OnSelect = func(it listbox.Items, i int) {
			previewCol := makeWidget(it.(fileItems)[i])
			w.MutateColViewState(func(s *colview.State) {
				s.Columns[2] = previewCol
			})
		}
	}

	w.MutateColViewState(func(s *colview.State) {
		*s = colview.State{
			Columns:     []el.Widget{parentCol, currentCol, previewCol},
			FocusColumn: 1,
		}
	})
}

// Selects nothing if the widget is a listbox.
func tryToSelectNothing(w el.Widget) {
	list, ok := w.(*listbox.Widget)
	if !ok {
		return
	}
	list.MutateListboxState(func(s *listbox.State) { s.Selected = -1 })
}

// Selects the item with the given name, if the widget is a listbox with
// fileItems and has such an item.
func tryToSelectName(w el.Widget, name string) {
	list, ok := w.(*listbox.Widget)
	if !ok {
		// Do nothing
		return
	}
	list.MutateListboxState(func(state *listbox.State) {
		items, ok := state.Items.(fileItems)
		if !ok {
			return
		}
		for i, file := range items {
			if file.Name() == name {
				state.Selected = i
			}
		}
	})
}

func makeWidget(f File) el.Widget {
	files, content, err := f.Read()
	if err != nil {
		return makeErrWidget(err)
	}

	if files != nil {
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})
		return &listbox.Widget{
			Padding:     1,
			ExtendStyle: true,
			State:       listbox.State{Items: fileItems(files)},
		}
	}

	lines := strings.Split(sanitize(string(content)), "\n")
	return &textview.Widget{
		State:      textview.State{Lines: lines},
		Scrollable: true,
	}
}

func makeErrWidget(err error) el.Widget {
	return layout.Label{Content: styled.MakeText(err.Error(), "red")}
}

type fileItems []File

func (it fileItems) Show(i int) styled.Text {
	// TODO: Support lsColors
	if it[i].Mode().IsDir() {
		return styled.MakeText(it[i].Name(), "blue")
	}
	return styled.Plain(it[i].Name())
}

func (it fileItems) Len() int { return len(it) }

func sanitize(content string) string {
	// Remove unprintable characters, and replace tabs with 4 spaces.
	var sb strings.Builder
	for _, r := range content {
		if r == '\t' {
			sb.WriteString("    ")
		} else if r == '\n' || unicode.IsGraphic(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}