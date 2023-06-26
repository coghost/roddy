package roddy

import (
	"fmt"
	"time"

	"github.com/coghost/xpretty"
	"github.com/pterm/pterm"
)

func SleepWithSpin(n int, args ...string) {
	msg := fmt.Sprintf("Wait for %d seconds before quitting ...", n)
	if len(args) > 0 {
		msg = args[0]
	}

	spinnerInfo, _ := pterm.DefaultSpinner.Start(xpretty.Yellow(msg))

	time.Sleep(time.Second * time.Duration(n))
	spinnerInfo.Info()
}
