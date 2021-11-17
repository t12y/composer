package composer_test

import (
	"reflect"
	"testing"

	"github.com/t12y/composer/composer"
)

func TestConfig_servicesToStart(t *testing.T) {
	tests := []struct {
		name        string
		services    map[string]composer.ServiceConfig
		serviceName []string
		want        []string
		wantErr     bool
	}{
		{
			name:        "one service",
			services:    map[string]composer.ServiceConfig{"s1": {Command: "echo 1"}},
			serviceName: []string{"s1"},
			want:        []string{"s1"},
		},
		{
			name: "simple dependency",
			services: map[string]composer.ServiceConfig{
				"s1": {Command: "echo 1", DependsOn: []string{"s2"}},
				"s2": {Command: "echo 2"},
			},
			serviceName: []string{"s1"},
			want:        []string{"s2", "s1"},
		},
		{
			name: "complex dependency",
			services: map[string]composer.ServiceConfig{
				"s1": {Command: "echo 1", DependsOn: []string{"s2", "s3"}},
				"s2": {Command: "echo 2", DependsOn: []string{"s3", "s4"}},
				"s3": {Command: "echo 3"},
				"s4": {Command: "echo 4"},
			},
			serviceName: []string{"s1"},
			want:        []string{"s4", "s3", "s2", "s1"},
		},
		{
			name: "more complex dependency",
			services: map[string]composer.ServiceConfig{
				"s1": {Command: "echo service 1", DependsOn: []string{"s2", "s3"}},
				"s2": {Command: "echo service 2", DependsOn: []string{"b1", "b2"}},
				"s3": {Command: "echo service 3", DependsOn: []string{"s2"}},
				"b1": {Command: "echo back-end 1"},
				"b2": {Command: "echo back-end 2"},
			},
			serviceName: []string{"s1"},
			want:        []string{"b2", "b1", "s2", "s3", "s1"},
		},
		{
			name: "init with two services",
			services: map[string]composer.ServiceConfig{
				"s1": {Command: "echo service 1", DependsOn: []string{"s2"}},
				"s2": {Command: "echo service 2", DependsOn: []string{"b1", "b2"}},
				"s3": {Command: "echo service 3", DependsOn: []string{"s2"}},
				"b1": {Command: "echo back-end 1"},
				"b2": {Command: "echo back-end 2"},
			},
			serviceName: []string{"s1", "s3"},
			want:        []string{"b2", "b1", "s2", "s3", "s1"},
		},
		{
			name: "circular dependency",
			services: map[string]composer.ServiceConfig{
				"s1": {Command: "echo 1", DependsOn: []string{"s2"}},
				"s2": {Command: "echo 2", DependsOn: []string{"s3"}},
				"s3": {Command: "echo 3", DependsOn: []string{"s1"}},
			},
			serviceName: []string{"s1"},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &composer.Config{
				Services: tt.services,
			}
			got, err := cfg.ServicesToStart(tt.serviceName...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ServicesToStart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServicesToStart() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironment_With(t *testing.T) {
	tests := []struct {
		name   string
		parent composer.Environment
		child  composer.Environment
		want   composer.Environment
	}{
		{
			name: "both nil",
			want: make(composer.Environment),
		},
		{
			name:  "parent nil",
			child: composer.Environment{"k": "v"},
			want:  composer.Environment{"k": "v"},
		},
		{
			name:   "child nil",
			parent: composer.Environment{"k": "v"},
			want:   composer.Environment{"k": "v"},
		},
		{
			name:   "collision",
			child:  composer.Environment{"k": "v1"},
			parent: composer.Environment{"k": "v2"},
			want:   composer.Environment{"k": "v1"},
		},
		{
			name:   "merge",
			child:  composer.Environment{"k1": "v1"},
			parent: composer.Environment{"k2": "v2"},
			want:   composer.Environment{"k1": "v1", "k2": "v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.child.Extends(tt.parent)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Environment.Extends() got = %v, want %v", got, tt.want)
			}
		})
	}
}
