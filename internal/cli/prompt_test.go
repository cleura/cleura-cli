package cli

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
)

func testPrompter(input string) *prompter {
	return &prompter{in: bufio.NewReader(strings.NewReader(input)), out: io.Discard, ttyFD: -1}
}

func TestPrompterLine(t *testing.T) {
	ctx := context.Background()

	// Multi-line piped input must survive across prompts: one shared buffered
	// reader, not one per prompt (regression: the first reader swallowed the
	// whole pipe).
	p := testPrompter("user\npass\n")
	if got, err := p.line(ctx, "Username"); err != nil || got != "user" {
		t.Fatalf("first line = %q, %v", got, err)
	}
	if got, err := p.line(ctx, "Password"); err != nil || got != "pass" {
		t.Fatalf("second line = %q, %v", got, err)
	}

	// A final line without a trailing newline is valid input.
	p = testPrompter("no-newline")
	if got, err := p.line(ctx, "Password"); err != nil || got != "no-newline" {
		t.Fatalf("unterminated line = %q, %v", got, err)
	}

	// Exhausted stdin names the prompt that went unanswered and wraps io.EOF.
	p = testPrompter("")
	_, err := p.line(ctx, "SMS code")
	if err == nil || !strings.Contains(err.Error(), "SMS code prompt") {
		t.Fatalf("EOF error = %v, want mention of the SMS code prompt", err)
	}

	// secret falls back to line reading when stdin is not a terminal.
	p = testPrompter("hunter2\n")
	if got, err := p.secret(ctx, "Password"); err != nil || got != "hunter2" {
		t.Fatalf("secret fallback = %q, %v", got, err)
	}

	// Piped secrets keep significant whitespace — only the line terminator
	// is stripped, matching the untrimmed interactive path.
	p = testPrompter(" pass word \r\n")
	if got, err := p.secret(ctx, "Password"); err != nil || got != " pass word " {
		t.Fatalf("secret should preserve spaces, got %q, %v", got, err)
	}
}

func TestConfirmNonTTYRefusesWithoutReading(t *testing.T) {
	// A non-TTY prompter must refuse (return false) without consuming stdin —
	// a piped "y" must not auto-confirm, and the piped data must survive for
	// the next read (e.g. a password).
	p := testPrompter("y\nsecret\n") // ttyFD == -1
	ok, err := p.confirm(context.Background(), "Replace it?")
	if err != nil || ok {
		t.Fatalf("non-TTY confirm = (%v,%v), want (false,nil)", ok, err)
	}
	// The "y" line must still be there to read.
	line, err := p.line(context.Background(), "Next")
	if err != nil || line != "y" {
		t.Fatalf("confirm consumed stdin: next line = %q, %v", line, err)
	}
}

func TestPrompterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A blocked read must not survive cancellation. Use a reader that never
	// returns data.
	blocked, _ := io.Pipe()
	p := &prompter{in: bufio.NewReader(blocked), out: io.Discard, ttyFD: -1}
	if _, err := p.line(ctx, "Username"); err != context.Canceled {
		t.Fatalf("cancelled line() = %v, want context.Canceled", err)
	}
}
