// prompt_test.go — table tests for loadPrompt and fill (design D4). Each
// case writes its text to a t.TempDir() file, so every case runs on every
// OS: no fake luvus and no Windows skip.
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writePrompt(t *testing.T, text string) string {
	t.Helper()
	abs := filepath.Join(t.TempDir(), "shot.md")
	if err := os.WriteFile(abs, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestLoadPrompt(t *testing.T) {
	tests := []struct {
		name string
		text string
		p    Prompt
		want error // nil, errEmptyPrompt, or any other error
	}{
		{
			name: "BOM is removed before the frontmatter split",
			text: "\ufeff---\nargument-hint: \"commit\"\n---\n\nbody\n",
			p:    Prompt{Body: "\nbody\n", Hint: "commit"},
		},
		{
			name: "no block: the body is the full text",
			text: "body line\nmore\n",
			p:    Prompt{Body: "body line\nmore\n"},
		},
		{
			name: "closed block: the body is the text after the closing line",
			text: "---\nargument-hint: fix\n---\nbody line\n",
			p:    Prompt{Body: "body line\n", Hint: "fix"},
		},
		{
			name: "unclosed block: there is no block, the body is the full text",
			text: "---\nargument-hint: fix\nbody line\n",
			p:    Prompt{Body: "---\nargument-hint: fix\nbody line\n"},
		},
		{
			name: "CRLF lines: the closing line and the hint match without the \\r",
			text: "---\r\nargument-hint: fix\r\n---\r\nbody line\r\n",
			p:    Prompt{Body: "body line\r\n", Hint: "fix"},
		},
		{
			name: "hint in double quotes",
			text: "---\nargument-hint: \"fix the thing\"\n---\nbody\n",
			p:    Prompt{Body: "body\n", Hint: "fix the thing"},
		},
		{
			name: "hint in single quotes",
			text: "---\nargument-hint: 'fix the thing'\n---\nbody\n",
			p:    Prompt{Body: "body\n", Hint: "fix the thing"},
		},
		{
			name: "hint without quotes",
			text: "---\nargument-hint: fix the thing\n---\nbody\n",
			p:    Prompt{Body: "body\n", Hint: "fix the thing"},
		},
		{
			name: "empty hint means no hint",
			text: "---\nargument-hint:\n---\nbody\n",
			p:    Prompt{Body: "body\n"},
		},
		{
			name: "hint missing means no hint",
			text: "---\nallowed-tools: Read\n---\nbody\n",
			p:    Prompt{Body: "body\n"},
		},
		{
			name: "block with no body is empty",
			text: "---\nargument-hint: fix\n---\n",
			want: errEmptyPrompt,
		},
		{
			name: "whitespace-only file is empty",
			text: "  \n\t\n",
			want: errEmptyPrompt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := loadPrompt(writePrompt(t, tt.text))
			if tt.want == nil {
				if err != nil {
					t.Fatalf("loadPrompt = %v, want nil", err)
				}
			} else if tt.want == errEmptyPrompt {
				if !errors.Is(err, errEmptyPrompt) {
					t.Fatalf("loadPrompt = %v, want errEmptyPrompt", err)
				}
				return
			} else {
				t.Fatalf("test %q has an unexpected want error: %v", tt.name, tt.want)
			}
			if p != tt.p {
				t.Errorf("loadPrompt = %#v, want %#v", p, tt.p)
			}
		})
	}
}

// A read failure returns the wrapped os.ReadFile error, not errEmptyPrompt.
func TestLoadPromptReadFailureIsNotErrEmptyPrompt(t *testing.T) {
	_, err := loadPrompt(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("loadPrompt(missing) = nil, want an error")
	}
	if errors.Is(err, errEmptyPrompt) {
		t.Errorf("loadPrompt(missing) = errEmptyPrompt, want the read error: %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("loadPrompt(missing) = %v, want a wrapped os.ErrNotExist", err)
	}
}

func TestFill(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		arg   string
		want  string
		empty bool // want errEmptyPrompt instead of a body
	}{
		{
			name: "plain placeholder",
			body: "Review $ARGUMENTS now.",
			arg:  "HEAD~1",
			want: "Review HEAD~1 now.",
		},
		{
			name: "default used",
			body: "Target: ${ARGUMENTS:-none}.",
			arg:  "",
			want: "Target: none.",
		},
		{
			name: "default not used",
			body: "Target: ${ARGUMENTS:-none}.",
			arg:  "my-change",
			want: "Target: my-change.",
		},
		{
			name: "value is not filled again",
			body: "Run $ARGUMENTS.",
			arg:  "$ARGUMENTS",
			want: "Run $ARGUMENTS.",
		},
		{
			name: "default is not filled again",
			body: "Target: ${ARGUMENTS:-$ARGUMENTS}.",
			arg:  "",
			want: "Target: $ARGUMENTS.",
		},
		{
			name: "other dollar text stays unchanged",
			body: "Diff $HOME vs !`clip-shot.sh`",
			arg:  "x",
			want: "Diff $HOME vs !`clip-shot.sh`",
		},
		{
			name: "send with no value removes $ARGUMENTS",
			body: "Review $ARGUMENTS now.",
			arg:  "",
			want: "Review  now.",
		},
		{
			name:  "ARGUMENTS-only body with an empty value is empty",
			body:  "$ARGUMENTS",
			arg:   "",
			empty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (Prompt{Body: tt.body}).fill(tt.arg)
			if tt.empty {
				if !errors.Is(err, errEmptyPrompt) {
					t.Fatalf("fill = %v, want errEmptyPrompt", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("fill = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("fill = %q, want %q", got, tt.want)
			}
		})
	}
}
