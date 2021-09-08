package main

import (
	"testing"
)

func Test_contains(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		str  string
		want bool
	}{
		{
			name: "does not contains string in array",
			arr:  []string{"testing", "example"},
			str:  "test",
			want: false,
		},
		{
			name: "contains string in array",
			arr:  []string{"test", "example"},
			str:  "test",
			want: true,
		},
		{
			name: "contains duplicate strings in array",
			arr:  []string{"test", "test"},
			str:  "test",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.arr, tt.str); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseTime(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "check empty",
			str:  "",
			want: "-",
		},
		{
			name: "check valid",
			str:  "2021-09-02T14:29:04.833Z",
			want: "Thu Sep 02 2021 2:29 PM",
		},
		{
			name: "check different format",
			str:  "2021-09-02T14:29:04Z07:00",
			want: "2021-09-02T14:29:04Z07:00",
		},
		{
			name: "check invalid format",
			str:  "2021-09-02",
			want: "2021-09-02",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTime(tt.str); got != tt.want {
				t.Errorf("parseTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
