package utils

import (
	"cmp"
	"errors"
	"fmt"
	"github.com/theckman/yacspin"
	"golang.org/x/term"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type SpinOptions struct {
	Label        string
	SuccessLabel string
	ErrorLabel   string
}

type SkippedError struct {
	Msg string
}

func (e SkippedError) Error() string {
	return e.Msg
}

func Spin(options SpinOptions, callback func() error) error {
	spinner, err := yacspin.New(yacspin.Config{
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[14],
		Prefix:            GetFullLine(options.Label, "", '─', 1),
		ColorAll:          true,
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
	})

	if spinner == nil || err != nil {
		return callback()
	}
	if err := spinner.Start(); err != nil {
		return callback()
	}

	callbackErr := callback()

	if errors.As(callbackErr, &SkippedError{}) {
		spinner.StopCharacter("⚠")
		spinner.Prefix(GetFullLine(options.Label, fmt.Sprintf("%v - SKIP ", callbackErr), '─', 1))
		_ = spinner.StopColors("fgYellow")
		_ = spinner.Stop()
		return nil
	} else if callbackErr != nil {
		spinner.Prefix(GetFullLine(cmp.Or(options.ErrorLabel, options.Label), fmt.Sprintf("%v - FAIL ", callbackErr), '─', 1))
		_ = spinner.StopFail()
	} else {
		spinner.Prefix(GetFullLine(cmp.Or(options.SuccessLabel, options.Label), "DONE ", '─', 1))
		_ = spinner.Stop()
	}
	return callbackErr
}

func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // Default value
	}
	return width
}
func GetFullLine(leftText string, rightText string, separator rune, offset int) string {
	strLen := utf8.RuneCountInString(leftText) + utf8.RuneCountInString(rightText)
	width := GetTerminalWidth() - offset
	center := " "
	if strLen < width {
		center = " " + strings.Repeat(string(separator), width-strLen-2) + " "
	}

	return leftText + center + rightText
}

func PrintSeparator(label string, separator rune) {
	width := GetTerminalWidth()
	labelLength := len(label)
	if labelLength == 0 {
		fmt.Println(strings.Repeat(string(separator), width))
		return
	}

	// Add spacing before and after label
	label = " " + label + " "
	labelLength = len(label)

	leftPadding := (width - labelLength) / 2
	rightPadding := width - leftPadding - labelLength
	fmt.Printf("%s%s%s\n",
		strings.Repeat(string(separator), leftPadding),
		label,
		strings.Repeat(string(separator), rightPadding),
	)
}
