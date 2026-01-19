package utils

import "github.com/common-nighthawk/go-figure"

func DrawBanner() {
	myFigure := figure.NewColorFigure("Cloud Doctor", "isometric3", "cyan", false)
	myFigure.Print()
}
