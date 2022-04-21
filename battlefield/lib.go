package battlefield

import (
	"fmt"
	"os"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

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

type BeforeAllHook func(cmd *cobra.Command)

func (f BeforeAllHook) apply(cmd *cobra.Command) {
	f(cmd)
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
