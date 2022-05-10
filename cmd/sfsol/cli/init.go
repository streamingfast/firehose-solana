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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var initCmd = &cobra.Command{Use: "init", Short: "Initializes StreamingFast's local environment", RunE: sfInitE}

func init() {
	RootCmd.AddCommand(initCmd)
}

func sfInitE(cmd *cobra.Command, args []string) (err error) {
	configFile := viper.GetString("global-config-file")
	zlog.Debug("starting init", zap.String("config-file", configFile))
	return nil
}
