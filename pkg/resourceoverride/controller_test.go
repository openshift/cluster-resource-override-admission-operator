package resourceoverride

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned/fake"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		options     *Options
		wantErr     bool
		wantName    string
		wantWorkers int
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: true,
		},
		{
			name: "nil client",
			options: &Options{
				ResyncPeriod: 10 * time.Minute,
				Workers:      1,
				Client:       nil,
			},
			wantErr: true,
		},
		{
			name: "valid options",
			options: &Options{
				ResyncPeriod: 10 * time.Minute,
				Workers:      2,
				Client: &operatorruntime.Client{
					Operator: fake.NewSimpleClientset(),
				},
			},
			wantErr:     false,
			wantName:    ControllerName,
			wantWorkers: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := New(test.options)
			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, c)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, c)
			require.Equal(t, test.wantName, c.Name())
			require.Equal(t, test.wantWorkers, c.WorkerCount())
			require.NotNil(t, c.Queue())
			require.NotNil(t, c.Informer())
			require.NotNil(t, c.Reconciler())
		})
	}
}
