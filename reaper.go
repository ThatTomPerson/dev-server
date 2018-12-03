// +build !windows

package main // import "ttp.sh/dev-server"

import reaper "github.com/ramr/go-reaper"

func Reaper() {
	reaper.Reap()
}
