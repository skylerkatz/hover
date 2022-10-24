package cmd

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	buildCmd "hover/cmd/build"
	commandCmd "hover/cmd/command"
	deployCmd "hover/cmd/deploy"
	secretCmd "hover/cmd/secret"
	stageCmd "hover/cmd/stage"
	"os"
)

var rootCmd = &cobra.Command{
	Version:       "0.0.1",
	Use:           "hover",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long:          `Deploy your Laravel applications serverlessly in AWS.`,
}

func Execute() {
	err := rootCmd.Execute()

	if err != nil {
		errorString := err.Error()

		fmt.Println("")
		pterm.Error.Println(errorString)

		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(stageCmd.Cmd())
	rootCmd.AddCommand(commandCmd.Cmd())
	rootCmd.AddCommand(secretCmd.Cmd())
	rootCmd.AddCommand(secretCmd.Cmd())
	rootCmd.AddCommand(deployCmd.Cmd())
	rootCmd.AddCommand(buildCmd.Cmd())

	rootCmd.SetVersionTemplate(pterm.FgMagenta.Sprint("HOVER") + " version " + pterm.FgYellow.Sprint("{{.Version}}") + "\n")

	rootCmd.SetHelpTemplate(`
` + pterm.FgYellow.Sprint("Description:") + `
  {{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)

	rootCmd.SetUsageTemplate(pterm.FgYellow.Sprint("Usage:") + `{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

` + pterm.FgYellow.Sprint("Commands:") + `{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + pterm.FgYellow.Sprint("Flags:") + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + pterm.FgYellow.Sprint("Global Flags:") + `
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`)
}
