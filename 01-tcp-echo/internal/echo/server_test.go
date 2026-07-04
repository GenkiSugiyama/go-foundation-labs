package echo

import (
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	input := strings.NewReader("hello\nworld\n")
	var output strings.Builder

	err := Handle(input, &output)
	if err != nil {
		t.Fatal(err)
	}

	want := "hello\nworld\n"
	got := output.String()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

}
