package composer

import (
	"reflect"
	"testing"
)

func Test_cleanCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  []string
		want []string
	}{
		{"empty", []string{}, []string{}},
		{"first empty", []string{" ", "cmd", "attr"}, []string{"cmd", "attr"}},
		{"middle empty", []string{"cmd", " ", "attr"}, []string{"cmd", "attr"}},
		{"last empty", []string{"cmd", "attr", "\t"}, []string{"cmd", "attr"}},
		{"command with spaces", []string{" cmd", "attr\t"}, []string{"cmd", "attr"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanCommand(tt.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cleanCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
