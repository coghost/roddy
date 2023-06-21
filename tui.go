package roddy

import (
	"time"

	"github.com/coghost/xpretty"
	"github.com/pterm/pterm"
)

func Spin(n int) {
	hint := xpretty.Yellowf("Wait for %d seconds before quitting ...", n)
	spinnerInfo, _ := pterm.DefaultSpinner.Start(hint)

	time.Sleep(time.Second * time.Duration(n))
	spinnerInfo.Info()
}
