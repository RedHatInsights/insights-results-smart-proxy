// Copyright 2021 Red Hat, Inc
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

package content

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommaSeparatedStrToTags(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.Equal(t, []string{}, commaSeparatedStrToTags(""))
	})

	t.Run("valid string", func(t *testing.T) {
		assert.Equal(t, []string{"a", "string"}, commaSeparatedStrToTags("a,string "))
	})
}

func TestTimeParse(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		_, missing, err := timeParse("")
		assert.True(t, missing)
		assert.NoError(t, err)
	})

	t.Run("valid layout", func(t *testing.T) {
		date, missing, err := timeParse("2021-02-03 01:02:03")
		assert.Equal(t, time.Date(2021, time.Month(2), 3, 1, 2, 3, 0, time.UTC), date)
		assert.False(t, missing)
		assert.NoError(t, err)
	})

	t.Run("invalid layout", func(t *testing.T) {
		_, missing, err := timeParse("2021/02/03 01:02:03")
		assert.False(t, missing)
		assert.Error(t, err)
	})
}
