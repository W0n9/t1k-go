package detection

import (
	"testing"
)

func TestResultBotDetectedFalse(t *testing.T) {
	r := &Result{}
	if r.BotDetected() {
		t.Error("expected BotDetected false for empty BotQuery")
	}
}

func TestResultBotDetectedTrue(t *testing.T) {
	r := &Result{BotQuery: []byte("some-bot-data")}
	if !r.BotDetected() {
		t.Error("expected BotDetected true when BotQuery non-empty")
	}
}

// BotDetected is driven by BotQuery only; BotBody alone does not trigger it.
func TestResultBotDetectedBotBodyOnly(t *testing.T) {
	r := &Result{BotBody: []byte("body-only")}
	if r.BotDetected() {
		t.Error("expected BotDetected false when only BotBody is set")
	}
}
