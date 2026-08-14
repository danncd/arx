/*
	Inline pinned-prompt layout, the shape of hermes' python CLI: the
	transcript flows top-down in the NORMAL buffer (it stays in the
	terminal after exit), a scroll region over rows 1..N-1 does the
	scrolling, and the input bar owns row N. The cursor is saved when
	leaving the transcript for the bar and restored on the way back, so
	printing always continues where the conversation left off.
*/

package cli

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type tui struct{ rows int }

/* True when stdout is a real terminal, not a pipe or file. */

func isTerminal() bool {
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

/* Terminal height via ioctl; 24 when unknowable. */

func termRows() int {
	var ws struct{ rows, cols, x, y uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.rows < 3 {
		return 24
	}
	return int(ws.rows)
}

func openTUI() *tui {
	t := &tui{rows: termRows()}
	fmt.Print("\033[2J\033[H")         // clear the visible screen, chat starts at the top
	fmt.Printf("\033[1;%dr", t.rows-1) // rows 1..N-1 scroll; row N is the bar
	fmt.Print("\033[H")                // cursor to row 1: the transcript grows from here
	return t
}

func (t *tui) close() {
	fmt.Print("\033[r")                     // release the scroll region
	fmt.Printf("\033[%d;1H\033[2K", t.rows) // clear the bar; the shell prompt takes over
}

/* Leaves the transcript for the bar: save the spot, draw the prompt. */

func (t *tui) prompt() {
	fmt.Print("\0337")
	fmt.Printf("\033[%d;1H\033[2K\033[7m > \033[0m ", t.rows)
}

/* Returns to the transcript where it left off. */

func (t *tui) transcript() {
	fmt.Print("\0338")
}
