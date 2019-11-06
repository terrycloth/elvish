// Package histlist implements the history listing addon.
package histlist

import (
	"fmt"
	"strings"

	"github.com/elves/elvish/cli"
	"github.com/elves/elvish/cli/el"
	"github.com/elves/elvish/cli/el/codearea"
	"github.com/elves/elvish/cli/el/combobox"
	"github.com/elves/elvish/cli/el/layout"
	"github.com/elves/elvish/cli/el/listbox"
	"github.com/elves/elvish/cli/histutil"
	"github.com/elves/elvish/styled"
)

// Config contains configurations to start history listing.
type Config struct {
	// Binding provides key binding.
	Binding el.Handler
	// Store provides the source of all commands.
	Store Store
}

// Store wraps the AllCmds method. It is a subset of histutil.Store.
type Store interface {
	AllCmds() ([]histutil.Entry, error)
}

var _ = Store(histutil.Store(nil))

// Start starts history listing.
func Start(app cli.App, cfg Config) {
	if cfg.Store == nil {
		app.Notify("no history store")
		return
	}
	cmds, err := cfg.Store.AllCmds()
	if err != nil {
		app.Notify("db error: " + err.Error())
	}

	w := combobox.New(combobox.Spec{
		CodeArea: codearea.Spec{Prompt: layout.ModePrompt("HISTLIST", true)},
		ListBox: listbox.Spec{
			OverlayHandler: cfg.Binding,
			OnAccept: func(it listbox.Items, i int) {
				text := it.(items)[i].Text
				app.CodeArea().MutateState(func(s *codearea.State) {
					buf := &s.Buffer
					if buf.Content == "" {
						buf.InsertAtDot(text)
					} else {
						buf.InsertAtDot("\n" + text)
					}
				})
				app.MutateState(func(s *cli.State) { s.Listing = nil })
			},
		},
		OnFilter: func(w combobox.Widget, p string) {
			it := filter(cmds, p)
			w.ListBox().Reset(it, it.Len()-1)
		},
	})

	app.MutateState(func(s *cli.State) { s.Listing = w })
	app.Redraw()
}

type items []histutil.Entry

func filter(allEntries []histutil.Entry, p string) items {
	if p == "" {
		return allEntries
	}
	var entries []histutil.Entry
	for _, entry := range allEntries {
		if strings.Contains(entry.Text, p) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (it items) Show(i int) styled.Text {
	// TODO: The alignment of the index works up to 10000 entries.
	return styled.Plain(fmt.Sprintf("%4d %s", it[i].Seq, it[i].Text))
}

func (it items) Len() int { return len(it) }