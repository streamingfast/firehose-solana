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

package dgraphql

import (
	"testing"

	"github.com/streamingfast/dgraphql"
	"github.com/streamingfast/sf-solana/dgraphql/resolvers"
	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	t.Skip("no schema any more ...")
	resolver, err := resolvers.NewRoot(nil, "", nil)
	require.NoError(t, err)

	// This makes the necessary parsing of all schemas to ensure resolver correctly
	// resolves the full schema.
	_, err = dgraphql.NewSchemas(resolver)
	require.NoError(t, err, "Invalid SOL schema nor resolver")
}
