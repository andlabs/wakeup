// 4 march 2014
package main

import (
	"fmt"
	"os"
//	"os/exec"
	"time"
	"github.com/andlabs/ui"
)

const (
	defCmdLine = "mpv -loop inf ~/ring.wav"
	defTime = "10:00 AM"
	timeFmt = "3:04 PM"
)

func myMain() {
	w := ui.NewWindow("wakeup", 400, 100)
	w.Closing = ui.Event()
	cmdbox := ui.NewLineEdit(defCmdLine)
	timebox := ui.NewLineEdit(defTime)
	bStart := ui.NewButton("Start")
	bStop := ui.NewButton("Stop")

	// a Stack to keep both buttons at the same size
	btnbox := ui.NewStack(ui.Horizontal, bStart, bStop)
	btnbox.SetStretchy(0)
	btnbox.SetStretchy(1)
	// and a Stack around that Stack to keep them at a reasonable size
	btnbox = ui.NewStack(ui.Horizontal, btnbox)

	// the main layout
	grid := ui.NewGrid(2,
		ui.NewLabel("Command"), cmdbox,
		ui.NewLabel("Time"), timebox,
		ui.Space(), ui.Space(),		// the Space on the right will consume the window blank space
		ui.Space(), btnbox)
	grid.SetStretchy(2, 1)			// make the Space noted above consume
	grid.SetFilling(0, 1)				// make the two textboxes grow horizontally
	grid.SetFilling(1, 1)

	err := w.Open(grid)
	if err != nil {
		ui.MsgBoxError("wakeup", "Error opening window: %v", err)
		os.Exit(1)
	}

mainloop:
	for {
		select {
		case <-w.Closing:
			break mainloop
		case <-bStart.Clicked:
			alarmTime, err := time.Parse(timeFmt, timebox.Text())
			if err != nil {
				ui.MsgBoxError("wakeup",
					"Error parsing time %q: %v\nMake sure your time is in the form %q (without quotes.",
					timebox.Text(), err, timeFmt)
				continue
			}
			fmt.Println(alarmTime, time.Now().Sub(alarmTime))
			// TODO
		case <-bStop.Clicked:
			// TODO
		}
	}
}

func main() {
	err := ui.Go(myMain)
	if err != nil {
		panic(fmt.Errorf("error initializing UI library: %v", err))
	}
}
