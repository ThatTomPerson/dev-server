// +build !windows

package main

import reaper "github.com/ramr/go-reaper"

func Reaper() {
	reaper.Reap()
}
