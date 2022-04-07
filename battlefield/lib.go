package battlefield

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

// Extract me into my own library, I've duplicated those at least 20 times now!

func copyFile(inPath, outPath string) {
	inFile, err := os.Open(inPath)
	NoError(err, "Unable to open actual file %q", inPath)
	defer inFile.Close()

	outFile, err := os.Create(outPath)
	NoError(err, "Unable to open expected file %q", outPath)
	defer outFile.Close()

	_, err = io.Copy(outFile, inFile)
	NoError(err, "Unable to copy file %q to %q", inPath, outPath)
}

func FileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		// For this script, we don't care
		return false
	}

	return !stat.IsDir()
}

func Ensure(condition bool, message string, args ...interface{}) {
	if !condition {
		Quit(message, args...)
	}
}

func NoError(err error, message string, args ...interface{}) {
	if err != nil {
		Quit(message+": "+err.Error(), args...)
	}
}

func Quit(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
	os.Exit(1)
}

type CommandOption interface {
	apply(cmd *cobra.Command)
}

type CommandOptionFunc func(cmd *cobra.Command)

func (f CommandOptionFunc) apply(cmd *cobra.Command) {
	f(cmd)
}

type Description string

func (d Description) apply(cmd *cobra.Command) {
	cmd.Long = strings.TrimSpace(dedent.Dedent(string(d)))
}

func Command(execute func(cmd *cobra.Command, args []string) error, usage, short string, opts ...CommandOption) CommandOption {
	return CommandOptionFunc(func(parent *cobra.Command) {
		parent.AddCommand(command(execute, usage, short, opts...))
	})
}

type BeforeAllHook func(cmd *cobra.Command)

func (f BeforeAllHook) apply(cmd *cobra.Command) {
	f(cmd)
}

func Run(usage, short string, opts ...CommandOption) {
	cmd := Root(usage, short, opts...)
	err := cmd.Execute()

	// FIXME: What is the right behavior on error from here?
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Root(usage, short string, opts ...CommandOption) *cobra.Command {
	beforeAllHook := BeforeAllHook(func(cmd *cobra.Command) {
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		if short != "" {
			cmd.Short = strings.TrimSpace(dedent.Dedent(short))
		}
	})

	return command(nil, usage, short, append([]CommandOption{beforeAllHook}, opts...)...)
}

func command(execute func(cmd *cobra.Command, args []string) error, usage, short string, opts ...CommandOption) *cobra.Command {
	command := &cobra.Command{}

	for _, opt := range opts {
		if _, ok := opt.(BeforeAllHook); ok {
			opt.apply(command)
		}
	}

	command.Use = usage
	command.Short = short
	command.RunE = execute

	for _, opt := range opts {
		switch opt.(type) {
		case BeforeAllHook:
			continue
		default:
			opt.apply(command)
		}
	}

	return command
}

func Dedent(input string) string {
	return strings.TrimSpace(dedent.Dedent(input))
}

func AskConfirmation(label string, args ...interface{}) (answeredYes bool, wasAnswered bool) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		wasAnswered = false
		return
	}

	prompt := promptui.Prompt{
		Label:     dedent.Dedent(fmt.Sprintf(label, args...)),
		IsConfirm: true,
	}

	_, err := prompt.Run()
	if err != nil {
		// zlog.Debug("unable to aks user to see diff right now, too bad", zap.Error(err))
		wasAnswered = false
		return
	}

	wasAnswered = true
	answeredYes = true

	return
}
