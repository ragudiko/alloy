package kubestate

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/stretchr/testify/require"
)

func TestKubeStateMetrics(t *testing.T) {
	tests := []struct {
		name    string
		args    Arguments
		wantErr bool
	}{
		{
			name: "default configuration",
			args: DefaultArguments,
		},
		{
			name: "with custom resources",
			args: Arguments{
				PollFrequency: "1m",
				PollTimeout:   "15s",
				Resources:     []string{"pods", "nodes"},
			},
		},
		{
			name: "with kubernetes client config",
			args: Arguments{
				PollFrequency: "1m",
				PollTimeout:   "15s",
				Client: kubernetes.ClientArguments{
					APIServer: "https://localhost:6443",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(component.Options{}, tt.args)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err = c.Run(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
} 