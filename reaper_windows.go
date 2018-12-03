// +build windows

package main

import "github.com/sirupsen/logrus"

func Reap() {
	logrus.Info("Reaping is not supported on windows")
}
