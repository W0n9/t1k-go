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
