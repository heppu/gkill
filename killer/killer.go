package killer

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/heppu/readline"
	"github.com/k0kubun/go-ansi"
	"github.com/mitchellh/go-ps"
)

type ByName []ps.Process

func (p ByName) Len() int           { return len(p) }
func (p ByName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByName) Less(i, j int) bool { return p[i].Executable() < p[j].Executable() }

type Killer struct {
	rl        *readline.Instance
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

	rl, err := readline.New("")
	if err != nil {
		return nil, err
	}
	rl.SetConfig(&readline.Config{
		HistoryLimit:        -1,
		FuncFilterInputRune: k.filterInput,
		Listener:            k,
		UniqueEditLine:      true,
		Stdout:              color.Output,
	})
	k.rl = rl
	return k, nil
}

func (k *Killer) Start() (err error) {
	for {
		if _, err = k.rl.Readline(); err != nil {
			if err == readline.ErrInterrupt {
				err = nil
				break
			}
			if err == io.EOF {
				err = nil
			}
			if err == nil {
				break
			}
		}
		if k.done {
			break
		}
	}

	k.rl.Clean()
	k.rl.Close()
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
	prompt := fmt.Sprintf("  %s", bold("Filter processes"))
	postPrompt := fmt.Sprintf(" (%d/%d) : ", len(k.filtered), len(k.processes))
	k.rl.SetPrompt(prompt + postPrompt)
	k.rl.Refresh()

	if len(k.filtered) > 0 {
		k.printProcesses()
		if !k.done {
			ansi.CursorPreviousLine(8)
			ansi.CursorForward(18 + len(postPrompt) + len(line))
		}
	}

	return nil, 0, false
}

func (k *Killer) printProcesses() {
	end := 7
	if len(k.filtered) < 7 {
		end = len(k.filtered)
	}
	var i int
	for i = 0; i < end; i++ {
		ansi.Println()
		ansi.EraseInLine(2)
		index := ((k.cursor+i)%len(k.filtered) + len(k.filtered)) % len(k.filtered)
		if i == end/2 {
			color.Set(color.FgCyan)
			ansi.Printf("❯ %s %d", k.filtered[index].Executable(), k.filtered[index].Pid())
			color.Unset()
		} else {
			ansi.Printf("  %s %d", k.filtered[index].Executable(), k.filtered[index].Pid())
		}
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
	case readline.CharEnter:
		k.done = true
		if len(k.filtered) > 0 {
			k.killProcess(syscall.SIGTERM)
		}
		return readline.CharInterrupt, true

	case readline.CharInterrupt:
		k.done = true
		return r, true

	case readline.CharNext:
		k.nextProcess()
		return readline.CharForward, true

	case readline.CharPrev:
		k.prevProcess()
		return readline.CharForward, true
	}
	return r, true
}
