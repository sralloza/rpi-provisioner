package info

import (
	"fmt"

	"github.com/gookit/color"
)

func Title(title string) {
	fmt.Printf("%s... ", title)
}

func Skipped() {
	color.FgCyan.Println("SKIPPED")
}

func Ok() {
	color.FgGreen.Println("OK")
}

func Fail() {
	color.FgRed.Println("FAIL")
}
