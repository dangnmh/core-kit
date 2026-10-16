package bootstrap_test

import (
	"errors"
	"testing"

	"github.com/dangnmh/core-kit/pkg/bootstrap"
	"github.com/stretchr/testify/assert"
)

func TestMustLoad(t *testing.T) {
	t.Parallel()

	type dummy struct {
		Val string
	}

	tests := []struct {
		name     string
		give     error
		wantCode int
		wantNil  bool
	}{
		{name: "ok", give: nil, wantCode: -1, wantNil: false},
		{name: "config error exits 78", give: errors.New("bad yaml"), wantCode: bootstrap.ExitConfig, wantNil: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code := -1
			res := bootstrap.MustLoad("svc", func() (*dummy, error) {
				if tt.give != nil {
					return nil, tt.give
				}
				return &dummy{Val: "loaded"}, nil
			}, func(c int) {
				code = c
			})

			assert.Equal(t, tt.wantCode, code)
			if tt.wantNil {
				assert.Nil(t, res)
			} else {
				assert.NotNil(t, res)
				assert.Equal(t, "loaded", res.Val)
			}
		})
	}
}
