package util

import (
	"github.com/fatih/color"
)

var (
	// ColorInfo returns a new function that returns info-colorized (green) strings for the
	// given arguments with fmt.Sprint().
	ColorInfo = color.New(color.FgGreen).SprintFunc()
)
