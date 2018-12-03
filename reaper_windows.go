// +build windows

package main

import "github.com/sirupsen/logrus"

func Reaper() {
	logrus.Info("Reaping is not supported on windows")
}
