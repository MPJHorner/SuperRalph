package progress

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressLatestIteration(t *testing.T) {
	tests := []struct {
		name    string
		entries []Entry
		want    int
	}{
		{
			name:    "empty",
			entries: []Entry{},
			want:    0,
		},
		{
			name: "single entry",
			entries: []Entry{
				{Iteration: 5},
			},
			want: 5,
		},
		{
			name: "multiple entries",
			entries: []Entry{
				{Iteration: 1},
				{Iteration: 2},
				{Iteration: 3},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Progress{Entries: tt.entries}
			assert.Equal(t, tt.want, p.LatestIteration())
		})
	}
}

func TestProgressLatestEntry(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p := &Progress{Entries: []Entry{}}
		assert.Nil(t, p.LatestEntry())
	})

	t.Run("with entries", func(t *testing.T) {
		entries := []Entry{
			{Iteration: 1, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Iteration: 2, Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
			{Iteration: 3, Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
		}
		p := &Progress{Entries: entries}
		got := p.LatestEntry()
		require.NotNil(t, got)
		assert.Equal(t, 3, got.Iteration)
	})
}
