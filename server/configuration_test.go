package main

import (
	"testing"
)

func Test_configuration_IsValid(t *testing.T) {
	type fields struct {
		HackeroneProgramHandle          string
		HackeroneApiIdentifier          string
		HackeroneApiKey                 string
		HackeronePollIntervalSeconds    int
		HackeroneSLAPollIntervalSeconds int
		HackeroneSLANew                 int
		HackeroneSLABounty              int
		HackeroneSLATriaged             int
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "empty",
			fields:  fields{},
			wantErr: true,
		},
		{
			name: "Handle only",
			fields: fields{
				HackeroneProgramHandle: "dummy",
			},
			wantErr: true,
		},
		{
			name: "API identifier and key only",
			fields: fields{
				HackeroneApiIdentifier: "dummyIdentifier",
				HackeroneApiKey:        "dummyKey",
			},
			wantErr: true,
		},
		{
			name: "Handle, API identifier and key only",
			fields: fields{
				HackeroneProgramHandle: "dummy",
				HackeroneApiIdentifier: "dummyIdentifier",
				HackeroneApiKey:        "dummyKey",
			},
			wantErr: true,
		},
		{
			name: "All valid configuration",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    30,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 1,
				HackeroneSLABounty:              1,
				HackeroneSLATriaged:             1,
			},
			wantErr: false,
		},
		{
			name: "invalid configuration (poll interval < min)",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    9,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 1,
				HackeroneSLABounty:              1,
				HackeroneSLATriaged:             1,
			},
			wantErr: true,
		},
		{
			name: "invalid configuration (poll interval > max)",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    3601,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 1,
				HackeroneSLABounty:              1,
				HackeroneSLATriaged:             1,
			},
			wantErr: true,
		},
		{
			name: "invalid configuration (sla new < min)",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    3600,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 0,
				HackeroneSLABounty:              1,
				HackeroneSLATriaged:             1,
			},
			wantErr: true,
		},
		{
			name: "invalid configuration (sla bounty < min)",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    3600,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 1,
				HackeroneSLABounty:              0,
				HackeroneSLATriaged:             1,
			},
			wantErr: true,
		},
		{
			name: "invalid configuration (sla triaged < min)",
			fields: fields{
				HackeroneProgramHandle:          "dummy",
				HackeroneApiIdentifier:          "dummyIdentifier",
				HackeroneApiKey:                 "dummyKey",
				HackeronePollIntervalSeconds:    3600,
				HackeroneSLAPollIntervalSeconds: 86400,
				HackeroneSLANew:                 1,
				HackeroneSLABounty:              1,
				HackeroneSLATriaged:             0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &configuration{
				HackeroneProgramHandle:          tt.fields.HackeroneProgramHandle,
				HackeroneApiIdentifier:          tt.fields.HackeroneApiIdentifier,
				HackeroneApiKey:                 tt.fields.HackeroneApiKey,
				HackeronePollIntervalSeconds:    tt.fields.HackeronePollIntervalSeconds,
				HackeroneSLAPollIntervalSeconds: tt.fields.HackeroneSLAPollIntervalSeconds,
				HackeroneSLANew:                 tt.fields.HackeroneSLANew,
				HackeroneSLABounty:              tt.fields.HackeroneSLABounty,
				HackeroneSLATriaged:             tt.fields.HackeroneSLATriaged,
			}
			if err := c.IsValid(); (err != nil) != tt.wantErr {
				t.Errorf("configuration.IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
