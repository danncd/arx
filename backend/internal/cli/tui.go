/*
	Inline pinned layout: a fixed Arx header on row 1, the transcript
	flowing top-down through a scroll region on rows 2..N-1 (normal
	buffer, so it stays in the terminal after exit), and the input bar
	on row N. The cursor is saved when leaving the transcript for the
	bar and restored on the way back. Height changes are picked up at
	prompt time, synchronously, so a redraw never races the stream.
*/

package cli

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type tui struct {
	rows, cols int
	header     string
}

/* True when stdout is a real terminal, not a pipe or file. */

func isTerminal() bool {
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

/* Terminal size via ioctl; 24x80 when unknowable. */

func termSize() (rows, cols int) {
	var ws struct{ rows, cols, x, y uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.rows < 4 || ws.cols < 8 {
		return 24, 80
	}
	return int(ws.rows), int(ws.cols)
}

func openTUI(header string) *tui {
	t := &tui{header: header}
	t.rows, t.cols = termSize()
	fmt.Print("\033[2J")
	t.layout()
	fmt.Print("\033[2;1H") // transcript grows from row 2
	return t
}

/* Draws the fixed rows and sets the scroll region for the current size. */

func (t *tui) layout() {
	fmt.Print("\033[r") // release any previous region before redrawing fixed rows
	title := " Arx · " + t.header
	if len(title) > t.cols {
		title = title[:t.cols]
	}
	pad := strings.Repeat(" ", t.cols-len(title))
	fmt.Printf("\033[1;1H\033[7m%s%s\033[0m", title, pad) // header bar, row 1
	fmt.Printf("\033[2;%dr", t.rows-1)                    // rows 2..N-1 scroll
}

/*
	Re-reads the terminal size before a prompt. On change the emulator
	has already reshuffled our rows into its own scrollback, and we
	keep no copy of the transcript to re-flow, so the honest move is a
	full reset: clear, redraw the fixed rows at the new geometry, and
	let the transcript restart. Earlier chat stays readable in the
	terminal's native scrollback.
*/

func (t *tui) refresh() {
	rows, cols := termSize()
	if rows == t.rows && cols == t.cols {
		return
	}
	t.rows, t.cols = rows, cols
	fmt.Print("\033[2J")
	t.layout()
	fmt.Print("\033[2;1H\033[90m(resized — earlier chat is in the scrollback above)\033[0m\n")
}

func (t *tui) close() {
	fmt.Print("\033[r")
	fmt.Printf("\033[%d;1H\033[2K", t.rows) // clear the bar; the shell takes over
}

/* Leaves the transcript for the bar: save the spot, draw the prompt. */

func (t *tui) prompt() {
	t.refresh()
	fmt.Print("\0337")
	fmt.Printf("\033[%d;1H\033[2K\033[7m > \033[0m ", t.rows)
}

/* Returns to the transcript where it left off. */

func (t *tui) transcript() {
	fmt.Print("\0338")
}
