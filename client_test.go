package prokishi

import "testing"

func TestFormatEngineLine(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		ver  string
		want string
	}{
		{
			name: "id name line gets prokishi suffix",
			cmd:  "id name Apery",
			ver:  "1.0",
			want: "id name Apery(prokishi 1.0)",
		},
		{
			name: "id author line gets fixed author suffix",
			cmd:  "id author YaneuraOu Developers",
			ver:  "1.0",
			want: "id author YaneuraOu Developers(secondarykey)",
		},
		{
			name: "non matching line passes through unchanged",
			cmd:  "usiok",
			ver:  "1.0",
			want: "usiok",
		},
		{
			name: "id name with empty version still appends parens",
			cmd:  "id name Foo",
			ver:  "",
			want: "id name Foo(prokishi )",
		},
		{
			name: "line that merely contains id name but not at index 0 is unchanged",
			cmd:  "info id name should not match",
			ver:  "1.0",
			want: "info id name should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEngineLine(tt.cmd, tt.ver)
			if got != tt.want {
				t.Errorf("formatEngineLine(%q, %q) = %q, want %q", tt.cmd, tt.ver, got, tt.want)
			}
		})
	}
}

func TestClientIsConnect(t *testing.T) {
	tests := []struct {
		name         string
		connectionId string
		want         bool
	}{
		{
			name:         "zero value connectionId is not connected",
			connectionId: "",
			want:         false,
		},
		{
			name:         "non empty connectionId is connected",
			connectionId: "some-uuid",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{connectionId: tt.connectionId}
			if got := cli.isConnect(); got != tt.want {
				t.Errorf("isConnect() = %v, want %v", got, tt.want)
			}
		})
	}
}

// NOTE: Run(), dial(), connect(), sendServer(), receiveUSI(), disconnect(),
// sendQuit(), and the stream-reading loop in receiveServer() all require a
// live gRPC connection to a prokishi-server and are intentionally out of
// scope for this phase.
