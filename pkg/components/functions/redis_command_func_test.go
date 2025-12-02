package functions

import (
	"reflect"
	"testing"
)

func Test_parseCommand(t *testing.T) {
	tests := []struct {
		name   string
		cmdStr string
		want   []string
	}{
		{
			name:   "Simple command",
			cmdStr: "PING",
			want:   []string{"PING"},
		},
		{
			name:   "Command with one arg",
			cmdStr: "GET mykey",
			want:   []string{"GET", "mykey"},
		},
		{
			name:   "Command with multiple args",
			cmdStr: "HINCRBY myhash field 1",
			want:   []string{"HINCRBY", "myhash", "field", "1"},
		},
		{
			name:   "Command with one quoted arg",
			cmdStr: `SET mykey "hello world"`,
			want:   []string{"SET", "mykey", "hello world"},
		},
		{
			name:   "Command with multiple mixed args",
			cmdStr: `HSET myhash field1 "value with spaces" field2 value2`,
			want:   []string{"HSET", "myhash", "field1", "value with spaces", "field2", "value2"},
		},
		{
			name:   "Quoted string at the beginning",
			cmdStr: `PUBLISH "a channel" "hello world"`,
			want:   []string{"PUBLISH", "a channel", "hello world"},
		},
		{
			name:   "Empty string",
			cmdStr: "",
			want:   []string{},
		},
		{
			name:   "String with only spaces",
			cmdStr: "   ",
			want:   []string{},
		},
		{
			name:   "Command with empty quoted string",
			cmdStr: `SET mykey ""`,
			want:   []string{"SET", "mykey", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCommand(tt.cmdStr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
