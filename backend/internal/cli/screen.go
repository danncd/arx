/*
	Fullscreen terminal mode: the alternate screen, with chat scrolling
	in rows 1..N-1 and the input bar pinned to row N. Only cursor
	placement lives here; printing stays ordinary fmt output, and the
	terminal's own scroll region does the scrolling.
*/

package cli

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type screen struct{ rows int }

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

func openScreen() *screen {
	s := &screen{rows: termRows()}
	fmt.Print("\033[?1049h")           // enter the alternate screen
	fmt.Print("\033[2J")               // clear it
	fmt.Printf("\033[1;%dr", s.rows-1) // rows 1..N-1 scroll; row N is pinned
	return s
}

func (s *screen) close() {
	fmt.Print("\033[r")      // release the scroll region
	fmt.Print("\033[?1049l") // back to the shell's scrollback
}

/*
	Puts the cursor at the bottom of the chat region. Output printed
	after this scrolls the region naturally; call it once per burst,
	not per token, or later prints overwrite earlier ones.
*/

func (s *screen) chat() {
	fmt.Printf("\033[%d;1H", s.rows-1)
}

/* Clears the input row, draws the bar, and leaves the cursor on it. */

func (s *screen) prompt() {
	fmt.Printf("\033[%d;1H\033[2K\033[7m > \033[0m ", s.rows)
}
