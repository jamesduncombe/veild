package veild

import (
	"errors"
	"testing"
)

func TestResolvers_NewResolvers(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     error
	}{
		{
			name:     "handle resolver file",
			filename: "fixtures/test_resolvers.yml",
			want:     nil,
		},
		{
			name:     "handle malformed file",
			filename: "fixtures/test_malformed_resolvers.yml",
			want:     ErrUnmarshallingResolvers,
		},
		{
			name:     "handle non-existant file",
			filename: "non-existant file",
			want:     ErrReadingResolversFile,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, got := NewResolvers(test.filename)
			if !errors.Is(got, test.want) {
				t.Errorf("wanted %v got %v", test.want, got)
			}
		})
	}
}
