// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/streamingfast/firehose-solana/types"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	// Needs to be in this file which is the main entry of wrapper binary
	_ "github.com/streamingfast/dauth/authenticator/gcp"
	_ "github.com/streamingfast/dauth/authenticator/null"   // auth null plugin
	_ "github.com/streamingfast/dauth/authenticator/secret" // auth secret/hard-coded plugin
	_ "github.com/streamingfast/dauth/ratelimiter/null"     // ratelimiter plugin

	"github.com/spf13/cobra"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dlauncher/flags"
	"github.com/streamingfast/dlauncher/launcher"
	_ "github.com/streamingfast/firehose-solana/types"
)

var RootCmd = &cobra.Command{Use: "firesol", Short: "Solana on StreamingFast"}
var allFlags = make(map[string]bool) // used as global because of async access to cobra init functions

func Main() {
	cobra.OnInitialize(func() {
		allFlags = flags.AutoBind(RootCmd, "SFSOL")
	})
	RootCmd.PersistentFlags().Bool("augmented-mode", false, "Setups binary to support augmented protobuf  blocks (pbsol.Block) which contains account changes, total ordering, etc...")
	RootCmd.PersistentFlags().StringP("data-dir", "d", "./sf-data", "Path to data storage for all components of the stack")
	RootCmd.PersistentFlags().StringP("config-file", "c", "./sf.yaml", "Configuration file to use. No config file loaded if set to an empty string.")
	RootCmd.PersistentFlags().String("validator-path", "solana-validator", "Path to the solana binary. Defaults to the solana found in your PATH")
	RootCmd.PersistentFlags().Bool("skip-checks", false, "Skip checks to ensure 'solana' binary is supported")
	RootCmd.PersistentFlags().String("log-format", "text", "Format for logging to stdout. Either 'text' or 'stackdriver'")
	RootCmd.PersistentFlags().Bool("log-to-file", true, "Also write logs to {sf-data-dir}/sf.log.json ")
	RootCmd.PersistentFlags().CountP("verbose", "v", "Enables verbose output (-vvvv for max verbosity)")
	RootCmd.PersistentFlags().String("log-level-switcher-listen-addr", "localhost:1065", "If non-empty, the process will listen on this address for json-formatted requests to change different logger levels (see DEBUG.md for more info)")
	RootCmd.PersistentFlags().String("metrics-listen-addr", MetricsListenAddr, "If non-empty, the process will listen on this address to server Prometheus metrics")
	RootCmd.PersistentFlags().String("pprof-listen-addr", "localhost:6060", "If non-empty, the process will listen on this address for pprof analysis (see https://golang.org/pkg/net/http/pprof/)")
	RootCmd.PersistentFlags().Duration("startup-delay", 0, "[DEV] Delay before launching actual application(s), useful to leave some time to perform maintenance operations, on persisten disks for example.")

	derr.Check("registering application flags", launcher.RegisterFlags(zlog, StartCmd))

	var availableCmds []string
	for app := range launcher.AppRegistry {
		availableCmds = append(availableCmds, app)
	}
	sort.Strings(availableCmds)

	StartCmd.SetHelpTemplate(fmt.Sprintf(startCmdHelpTemplate, strings.Join(availableCmds, "\n  ")))
	StartCmd.Example = startCmdExample

	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := setupCmd(cmd); err != nil {
			return fmt.Errorf("failed to bootstrap system")
		}

		startupDelay := viper.GetDuration("global-startup-delay")
		if startupDelay.Microseconds() > 0 {
			zlog.Info("sleeping before starting application(s)", zap.Duration("delay", startupDelay))
			time.Sleep(startupDelay)
		}

		augmentedMode := viper.GetBool("global-augmented-mode")
		zlog.Info("setting up firesol bstream", zap.Bool("augmented", augmentedMode))
		if augmentedMode {
			types.SetupSfSolAugmented()
		}

		return nil
	}

	derr.Check("Solana on StreamingFast", RootCmd.Execute())
}

var startCmdExample = `firesol start relayer merger --merger-grpc-serving-addr=localhost:12345 --relayer-merger-addr=localhost:12345`
var startCmdHelpTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}} [all|command1 [command2...]]{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}

Available applications:
  %s{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
