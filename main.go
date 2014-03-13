// 4 march 2014
package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
	"github.com/andlabs/ui"
)

const (
	defCmdLine = "mpv -loop inf ~/ring.wav"
	defTime = "10:30 AM"
	timeFmt = "3:04 PM"
)

// If later hasn't happened yet, make it happen on the day of now; if not, the day after.
func bestTime(now time.Time, later time.Time) time.Time {
	now = now.Local()		// use local time to make things make sense
	nowh, nowm, nows := now.Clock()
	laterh, laterm, laters := later.Clock()
	add := false
	if nowh > laterh {
		add = true
	} else if (nowh == laterh) && (nowm > laterm) {
		add = true
	} else if (nowh == laterh) && (nowm == laterm) && (nows >= laters) {
		// >= in the case we're on the exact second; add a day because the alarm should have gone off by now otherwise!
		add = true
	}
	if add {
		now = now.AddDate(0, 0, 1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(),
		laterh, laterm, laters, 0,
		now.Location())
}

func myMain() {
	var cmd *exec.Cmd
	var timer *time.Timer
	var timerChan <-chan time.Time

	stop := func() {
		if cmd != nil {		// stop the command if it's running
			err := cmd.Process.Kill()
			if err != nil {
				ui.MsgBoxError(
					fmt.Sprintf("Error killing process: %v", err),
					"You may need to kill it manually.")
			}
			err = cmd.Process.Release()
			if err != nil {
				ui.MsgBoxError(
					fmt.Sprintf("Error releasing process: %v", err),
					"")
			}
			cmd = nil
		}
		if timer != nil {		// stop the timer if we started it
			timer.Stop()
			timer = nil
			timerChan = nil
		}
	}

	w := ui.NewWindow("wakeup", 400, 100)
	w.Closing = ui.Event()
	cmdbox := ui.NewLineEdit(defCmdLine)
	timebox := ui.NewLineEdit(defTime)
	bStart := ui.NewButton("Start")
	bStop := ui.NewButton("Stop")

	// a Stack to keep both buttons at the same size
	btnbox := ui.NewHorizontalStack(bStart, bStop)
	btnbox.SetStretchy(0)
	btnbox.SetStretchy(1)
	// and a Stack around that Stack to keep them at a reasonable size, with space to their right
	btnbox = ui.NewHorizontalStack(btnbox)

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
		ui.MsgBoxError(
			fmt.Sprintf("Error opening window: %v", err),
			"")
		os.Exit(1)
	}

mainloop:
	for {
		select {
		case <-w.Closing:
			break mainloop
		case <-bStart.Clicked:
			stop()		// only one alarm at a time
			alarmTime, err := time.Parse(timeFmt, timebox.Text())
			if err != nil {
				ui.MsgBoxError(
					fmt.Sprintf("Error parsing time %q: %v", timebox.Text(), err),
					fmt.Sprintf("Make sure your time is in the form %q (without quotes).", timeFmt))
				continue
			}
			now := time.Now()
			later := bestTime(now, alarmTime)
			timer = time.NewTimer(later.Sub(now))
			timerChan = timer.C
		case <-timerChan:
			cmd = exec.Command("/bin/sh", "-c", "exec " + cmdbox.Text())
			// keep stdin /dev/null in case user wants to run multiple alarms on one instance (TODO should I allow this program to act as a pipe?)
			// keep stdout /dev/null to avoid stty mucking
			cmd.Stderr = os.Stderr
			err := cmd.Start()
			if err != nil {
				ui.MsgBoxError(
					fmt.Sprintf("Error running program: %v", err),
					"")
				cmd = nil
			}
			timer = nil
			timerChan = nil
		case <-bStop.Clicked:
			stop()
		}
	}

	// clean up
	stop()
}

func main() {
	err := ui.Go(myMain)
	if err != nil {
		panic(fmt.Errorf("error initializing UI library: %v", err))
	}
}
