package info

import (
	"fmt"

	"github.com/gookit/color"
)

func Title(title string, args ...any) {
	if len(args) > 0 {
		title = fmt.Sprintf(title, args...)
	}
	fmt.Printf("%s... ", title)
}

func Skipped() {
	color.FgCyan.Println("SKIPPED")
	fmt.Println()
}

func Ok() {
	color.FgGreen.Println("OK")
	fmt.Println()
}

func Fail() {
	color.FgRed.Println("FAIL")
	fmt.Println()
}
