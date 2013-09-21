package edit

import (
	"os"
	"bufio"
	"../async"
)

type Key struct {
	rune
	Ctrl bool
	Alt bool
}

func PlainKey(r rune) Key {
	return Key{rune: r}
}

func CtrlKey(r rune) Key {
	return Key{rune: r, Ctrl: true}
}

func (k Key) String() (s string) {
	if k.Ctrl {
		s += "Ctrl-"
	}
	if k.Alt {
		s += "Alt-"
	}
	s += string(k.rune)
	return
}

const (
	F1 rune = -1-iota
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	F9
	F10
	F11
	F12

	Escape // ^[
	Backspace // ^?

	Up // ^[OA
	Down // ^[OB
	Right // ^[OC
	Left // ^[OD

	Home // ^[[1~
	Insert // ^[[2~
	Delete // ^[[3~
	End // ^[[4~
	PageUp // ^[[5~
	PageDown // ^[[6~
)

// reader is the part of an Editor responsible for reading and decoding
// terminal key sequences.
type reader struct {
	runeReader *async.RuneReader
	readAhead []Key
}

func newReader(f *os.File) *reader {
	return &reader{
		async.NewRuneReader(bufio.NewReaderSize(f, 0)),
		make([]Key, 0),
	}
}

// type readerState func(rune) (bool, readerState)

func (rd *reader) readKey() (k Key, err error) {
	if n := len(rd.readAhead); n > 0 {
		k = rd.readAhead[0]
		rd.readAhead = rd.readAhead[1:]
		return
	}

	rd.runeReader.Go <- true
	item := <-rd.runeReader.Items

	if err = item.Err; err != nil {
		return
	}

	switch r := item.rune; r {
	case 0x0:
		k = CtrlKey('`')
	case 0x1d:
		k = CtrlKey('6')
	case 0x1f:
		k = CtrlKey('/')
	case 0x7f:
		k = PlainKey(Backspace)
	/*
	case 0x1b:
		// ^[, or Escape
		k = CtrlKey('[')
	*/
	default:
		if 0x1 <= r && r <= 0x1d {
			k = CtrlKey(r+0x40)
		} else {
			k = PlainKey(r)
		}
	}
	return
}
