package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// prompter reads interactive input. One prompter must be shared across all
// prompts of a command invocation: it owns a single buffered reader, so piped
// multi-line input survives from one prompt to the next.
type prompter struct {
	in    *bufio.Reader
	out   io.Writer // prompts and echoes; stderr so stdout stays data-only
	ttyFD int       // -1 when stdin is not a terminal
}

func newPrompter(cmd *cobra.Command) *prompter {
	p := &prompter{
		in:    bufio.NewReader(cmd.InOrStdin()),
		out:   cmd.ErrOrStderr(),
		ttyFD: -1,
	}
	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		p.ttyFD = int(f.Fd())
	}
	return p
}

// line prompts for one line of input, trimmed of surrounding whitespace.
// A final line without a trailing newline is accepted; an exhausted stdin
// yields an error that names the unanswered prompt (and wraps io.EOF so
// callers can add context).
func (p *prompter) line(ctx context.Context, name string) (string, error) {
	return p.read(ctx, name, strings.TrimSpace)
}

func (p *prompter) read(ctx context.Context, name string, clean func(string) string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", name)

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		l, err := p.in.ReadString('\n')
		ch <- result{l, err}
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(p.out)
		return "", ctx.Err()
	case r := <-ch:
		line := clean(r.line)
		if r.err != nil {
			// ReadString returns the partial line together with io.EOF; input
			// like `printf '%s' "$PASSWORD"` (no trailing newline) is valid.
			if errors.Is(r.err, io.EOF) {
				if line != "" {
					return line, nil
				}
				return "", fmt.Errorf("stdin closed before the %s prompt was answered: %w", name, r.err)
			}
			return "", fmt.Errorf("reading %s: %w", name, r.err)
		}
		return line, nil
	}
}

// confirm asks a yes/no question, defaulting to no. When stdin is not a
// terminal it refuses without reading: there is no one to answer, and reading
// would consume piped data meant for a later prompt (e.g. a password), and a
// piped "y" must never auto-confirm a destructive action.
func (p *prompter) confirm(ctx context.Context, question string) (bool, error) {
	if p.ttyFD < 0 {
		return false, nil
	}
	answer, err := p.read(ctx, question+" [y/N]", strings.TrimSpace)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

// secret reads without echo when stdin is a terminal and falls back to line
// otherwise (piped input). On cancellation the terminal state is restored
// explicitly: term.ReadPassword's own restore never runs if the process is
// interrupted while it blocks, which would leave the user's shell with echo
// disabled.
func (p *prompter) secret(ctx context.Context, name string) (string, error) {
	if p.ttyFD < 0 {
		// Piped credentials: strip only the line terminator. Interior and
		// trailing spaces may be significant in a password, and the
		// interactive path (term.ReadPassword) does not trim either.
		return p.read(ctx, name, func(s string) string { return strings.TrimRight(s, "\r\n") })
	}

	state, err := term.GetState(p.ttyFD)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(p.out, "%s: ", name)

	type result struct {
		b   []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		b, err := term.ReadPassword(p.ttyFD)
		ch <- result{b, err}
	}()

	select {
	case <-ctx.Done():
		_ = term.Restore(p.ttyFD, state)
		fmt.Fprintln(p.out)
		return "", ctx.Err()
	case r := <-ch:
		fmt.Fprintln(p.out)
		if r.err != nil {
			return "", fmt.Errorf("reading %s: %w", name, r.err)
		}
		return string(r.b), nil
	}
}
