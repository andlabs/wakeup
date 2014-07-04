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
	defTime    = "10:30 AM"
	timeFmt    = "3:04 PM"
)

// If later hasn't happened yet, make it happen on the day of now; if not, the day after.
func bestTime(now time.Time, later time.Time) time.Time {
	now = now.Local() // use local time to make things make sense
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

type MainWindow struct {
	cmd      *exec.Cmd
	stopChan chan struct{}

	win     *ui.Window
	cmdbox  *ui.LineEdit
	timebox *ui.LineEdit
	bStart  *ui.Button
	bStop   *ui.Button
	status  *ui.Label
}

// this is run as a separate goroutine
// mw.stopChan must be valid before this function starts and must be closed after this function returns
func (mw *MainWindow) timer(t time.Duration) {
	timer := time.NewTimer(t)
	for {
		select {
		case <-timer.C:
			// send a signal to the main window that we're ready to run the command it has
			ui.Post(mw.win, nil) // no data needed; just a signal
			return
		case <-mw.stopChan:
			timer.Stop()
			return
		}
	}
	panic("unreachable") // just in case
}

// this is called by mw.Event() when we need to stop the alarm
// it must run on the same OS thread as mw.Event()
func (mw *MainWindow) stop() {
	if mw.cmd != nil { // stop the command if it's running
		err := mw.cmd.Process.Kill()
		if err != nil {
			mw.win.MsgBoxError(
				fmt.Sprintf("Error killing process: %v", err),
				"You may need to kill it manually.")
		}
		err = mw.cmd.Process.Release()
		if err != nil {
			mw.win.MsgBoxError(
				fmt.Sprintf("Error releasing process: %v", err),
				"")
		}
		mw.cmd = nil
	}
	if mw.stopChan != nil { // stop the timer if it's still running
		mw.stopChan <- struct{}{}
		close(mw.stopChan)
		mw.stopChan = nil
	}
	mw.status.SetText("")
}

func NewMainWindow() (mw *MainWindow) {
	mw = new(MainWindow)

	mw.win = ui.NewWindow("wakeup", 400, 100)
	mw.win.Closing = mw.Closing
	mw.win.Posted = mw.Fire
	mw.cmdbox = ui.NewLineEdit(defCmdLine)
	mw.timebox = ui.NewLineEdit(defTime)
	mw.bStart = ui.NewButton("Start")
	mw.bStart.Clicked = mw.StartClicked
	mw.bStop = ui.NewButton("Stop")
	mw.bStop.Clicked = mw.stop
	mw.status = ui.NewLabel("")

	// a Stack to keep both buttons at the same size
	btnbox := ui.NewHorizontalStack(mw.bStart, mw.bStop)
	btnbox.SetStretchy(0)
	btnbox.SetStretchy(1)
	// and a Stack around that Stack to keep them at a reasonable size, with space to their right
	btnbox = ui.NewHorizontalStack(btnbox, mw.status)

	// the main layout
	grid := ui.NewGrid(2,
		ui.NewLabel("Command"), mw.cmdbox,
		ui.NewLabel("Time"), mw.timebox,
		ui.Space(), ui.Space(), // the Space on the right will consume the window blank space
		ui.Space(), btnbox)
	grid.SetStretchy(2, 1) // make the Space noted above consume
	grid.SetFilling(0, 1)  // make the two textboxes grow horizontally
	grid.SetFilling(1, 1)
	grid.SetFilling(0, 0) // make the two Labels fill their cells so alignment works right on GTK+ (TODO)
	grid.SetFilling(1, 0)

	mw.win.Open(grid)

	return mw
}

func (mw *MainWindow) Closing() bool {
	mw.stop()
	ui.Stop <- struct{}{}
	return true
}

func (mw *MainWindow) StartClicked() {
	mw.stop() // only one alarm at a time
	alarmTime, err := time.Parse(timeFmt, mw.timebox.Text())
	if err != nil {
		mw.win.MsgBoxError(
			fmt.Sprintf("Error parsing time %q: %v", mw.timebox.Text(), err),
			fmt.Sprintf("Make sure your time is in the form %q (without quotes).", timeFmt))
		return
	}
	now := time.Now()
	later := bestTime(now, alarmTime)
	mw.stopChan = make(chan struct{})
	go mw.timer(later.Sub(now))
	mw.status.SetText("Started")
}

func (mw *MainWindow) Fire(data interface{}) {
	mw.cmd = exec.Command("/bin/sh", "-c", "exec "+mw.cmdbox.Text())
	// keep stdin /dev/null in case user wants to run multiple alarms on one instance (TODO should I allow this program to act as a pipe?)
	// keep stdout /dev/null to avoid stty mucking
	mw.cmd.Stderr = os.Stderr
	err := mw.cmd.Start()
	mw.status.SetText("Firing")
	if err != nil {
		mw.win.MsgBoxError(
			fmt.Sprintf("Error running program: %v", err),
			"")
		mw.cmd = nil
		mw.status.SetText("")
	}
	// we're done with the timer, but the goroutine that handles it has returned (or will after we do)
	// so close the stopChan now so that the next call to mw.stop() doesn't hang or crash
	close(mw.stopChan)
	mw.stopChan = nil
}

func main() {
	err := ui.Go(func() { // start
		NewMainWindow()
	})
	if err != nil {
		panic(fmt.Errorf("error initializing UI library: %v", err))
	}
}
