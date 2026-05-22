package dashboard

import (
	"strings"
	"testing"
)

// parseSSEFrames ist der Test-Helper für Sprint-17-SSE-Endpoints (S17-8).
// Nimmt einen rohen text/event-stream-Body (z.B. aus httptest.ResponseRecorder.Body)
// und parsed die Frames raus.
//
// Ein Frame ist alles bis zum nächsten "\n\n". Innerhalb eines Frames:
//   - event: <name>   → Frame-Type (default "message")
//   - data: <payload> → Frame-Payload (mehrere data:-Zeilen werden mit \n joined)
//
// Heartbeat-Frames (event: ping) werden mit Type="ping" zurückgegeben — der
// aufrufende Test entscheidet, ob er sie ignoriert.
//
// Beispiel:
//
//	rec := httptest.NewRecorder()
//	handler.ServeHTTP(rec, req)
//	frames := parseSSEFrames(rec.Body.String())
//	for _, f := range frames {
//	    if f.Type == "ping" { continue }
//	    require.JSONEq(t, `{"id":"abc"}`, f.Data)
//	}
type sseFrame struct {
	Type string
	Data string
}

func parseSSEFrames(raw string) []sseFrame {
	var frames []sseFrame
	for _, block := range strings.Split(raw, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		frame := sseFrame{Type: "message"}
		var dataLines []string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				frame.Type = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}
		frame.Data = strings.Join(dataLines, "\n")
		frames = append(frames, frame)
	}
	return frames
}

// TestParseSSEFrames sichert das Test-Helper-Verhalten. Wenn sich das
// Frame-Format ändert (z.B. multi-line data:), brechen diese Tests, bevor sie
// auf irreführende SSE-Endpoint-Tests wirken.
func TestParseSSEFrames(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []sseFrame
	}{
		{
			name: "single data frame",
			raw:  "data: hello\n\n",
			want: []sseFrame{{Type: "message", Data: "hello"}},
		},
		{
			name: "ping heartbeat",
			raw:  "event: ping\ndata: {}\n\n",
			want: []sseFrame{{Type: "ping", Data: "{}"}},
		},
		{
			name: "mixed stream",
			raw:  "data: first\n\nevent: ping\ndata: {}\n\ndata: second\n\n",
			want: []sseFrame{
				{Type: "message", Data: "first"},
				{Type: "ping", Data: "{}"},
				{Type: "message", Data: "second"},
			},
		},
		{
			name: "empty",
			raw:  "",
			want: nil,
		},
		{
			name: "DONE marker",
			raw:  "data: {\"x\":1}\n\ndata: [DONE]\n\n",
			want: []sseFrame{
				{Type: "message", Data: `{"x":1}`},
				{Type: "message", Data: "[DONE]"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSSEFrames(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("frame count: want %d, got %d (%v)", len(tc.want), len(got), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("frame %d: want %+v, got %+v", i, tc.want[i], got[i])
				}
			}
		})
	}
}
