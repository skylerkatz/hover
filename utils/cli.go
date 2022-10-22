package utils

import (
	"fmt"
	"github.com/pterm/pterm"
	"os"
	"os/exec"
	"strings"
)

func PrintSuccess(text string) {
	fmt.Println(pterm.BgGreen.Sprint(" DONE ") + " " + pterm.FgGreen.Sprint(text))
}

func PrintInfo(text string) {
	pterm.Info.Println(text)
}

func PrintWarning(text string) {
	pterm.Warning.Println(text)
}

func PrintStep(text string) {
	fmt.Println(pterm.BgMagenta.Sprint(" STEP ") + " " + pterm.FgMagenta.Sprint(text))
}

func Exec(command string, cwd string) error {
	commandString := strings.Fields(command)

	cmd := exec.Command(commandString[0], commandString[1:]...)

	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
