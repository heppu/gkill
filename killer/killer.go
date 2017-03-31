package killer

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/heppu/go-ps"
	"github.com/heppu/rawterm"
	"github.com/k0kubun/go-ansi"
)

type ByName []ps.Process

func (p ByName) Len() int           { return len(p) }
func (p ByName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByName) Less(i, j int) bool { return p[i].Executable() < p[j].Executable() }

type Killer struct {
	rt        *rawterm.Instance
	processes []ps.Process
	filtered  []ps.Process
	cursor    int
	filter    string
	done      bool
	killed    bool
	err       error
}

func NewKiller() (*Killer, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	if len(processes) == 0 {
		return nil, fmt.Errorf("No processes")
	}
	sort.Sort(ByName(processes))

	k := &Killer{
		processes: processes,
		filtered:  processes,
		cursor:    len(processes) / 2,
	}

	color.Output = ansi.NewAnsiStdout()

	rt, err := rawterm.New("")
	if err != nil {
		return nil, err
	}
	rt.SetConfig(&rawterm.Config{
		FuncFilterInputRune: k.filterInput,
		Listener:            k,
		UniqueEditLine:      true,
		Stdout:              color.Output,
	})
	k.rt = rt
	return k, nil
}

func (k *Killer) Start() (err error) {
	for {
		if _, err = k.rt.Readline(); err != nil {
			if err == rawterm.ErrInterrupt {
				err = nil
				break
			}
			if err == io.EOF {
				err = nil
				break
			}
		}
		if k.done {
			break
		}
	}
	// TODO: Figure out why we need this sleep
	// to make clean exit
	time.Sleep(time.Millisecond * 10)
	k.rt.Close()
	err = k.err
	return
}

func (k *Killer) nextProcess() {
	if len(k.filtered) > 1 {
		k.cursor = ((k.cursor+1)%len(k.filtered) + len(k.filtered)) % len(k.filtered)
	}
}

func (k *Killer) prevProcess() {
	if len(k.filtered) > 1 {
		k.cursor = ((k.cursor-1)%len(k.filtered) + len(k.filtered)) % len(k.filtered)
	}
}

func (k *Killer) killProcess(sig syscall.Signal) {
	var p *os.Process
	if k.filtered[k.cursor].Pid() == os.Getpid() {
		return
	}
	if p, k.err = os.FindProcess(k.filtered[k.cursor].Pid()); k.err != nil {
		return
	}

	k.err = p.Signal(sig)
}

func (k *Killer) OnChange(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	if !k.done {
		k.filter = string(line)
		k.filterProcesses()
	}

	bold := color.New(color.Bold).SprintFunc()
	prompt := "  Filter processes"
	postPrompt := fmt.Sprintf(" (%d/%d)", len(k.filtered), len(k.processes))
	if len(k.filtered) > 0 {
		k.rt.SetPrompt(bold(prompt) + color.GreenString(postPrompt) + bold(": "))
	} else {
		k.rt.SetPrompt(bold(prompt) + color.RedString(postPrompt) + bold(": "))
	}
	k.rt.Refresh()

	k.printProcesses()
	if !k.done {
		ansi.CursorPreviousLine(8)
		ansi.CursorForward(len(prompt) + len(postPrompt) + len(line) + 2)
	}

	return nil, 0, false
}

func (k *Killer) printProcesses() {
	end := 7
	faint := color.New(color.Faint).SprintFunc()
	if len(k.filtered) < 7 {
		end = len(k.filtered)
	}
	var i int
	for i = 0; i < end; i++ {
		ansi.Println()
		ansi.EraseInLine(2)
		index := ((k.cursor+i-(end/2))%len(k.filtered) + len(k.filtered)) % len(k.filtered)
		name := k.filtered[index].Executable()
		user := k.filtered[index].User()
		pid := fmt.Sprintf("%d", k.filtered[index].Pid())
		if i == end/2 {
			color.Set(color.FgCyan)
			if k.killed {
				color.Set(color.FgRed)
			}
			ansi.Print("❯")
		} else {
			ansi.Print(" ")
		}
		ansi.Printf(" %s", name)
		ansi.CursorForward(17 - len(name))
		ansi.Printf("%s", faint(pid))
		ansi.CursorForward(8 - len(pid))
		ansi.Printf("%s", faint(user))
		color.Unset()
	}
	if end == 0 {
		ansi.Println()
		ansi.EraseInLine(2)
		ansi.Printf(color.RedString("  No results..."))
		i++
	}
	for ; i < 8; i++ {
		ansi.Println()
		ansi.EraseInLine(2)
	}
}

func (k *Killer) filterProcesses() {
	var found bool
	oldPid := 0
	if len(k.filtered) > k.cursor {
		oldPid = k.filtered[k.cursor].Pid()
	}
	k.filtered = make([]ps.Process, 0)

	for i := 0; i < len(k.processes); i++ {
		if index := strings.Index(strings.ToUpper(k.processes[i].Executable()), strings.ToUpper(k.filter)); index != -1 || k.filter == "" {
			if !found && oldPid == k.processes[i].Pid() {
				k.cursor = len(k.filtered)
				found = true
			}

			k.filtered = append(k.filtered, k.processes[i])
		}
	}

	if !found {
		k.cursor = 0
	}
}

func (k *Killer) filterInput(r rune) (rune, bool) {
	switch r {
	case rawterm.CharEnter:
		k.done = true
		if len(k.filtered) > 0 {
			k.killed = true
			k.killProcess(syscall.SIGTERM)
		}
		return rawterm.CharInterrupt, true

	case rawterm.CharInterrupt:
		k.done = true
		return r, true

	case rawterm.CharNext:
		k.nextProcess()
		return rawterm.CharForward, true

	case rawterm.CharPrev:
		k.prevProcess()
		return rawterm.CharForward, true
	}
	return r, true
}
