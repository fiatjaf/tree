package main

import (
	"bytes"
	"testing"

	"github.com/fiatjaf/tree/ostree"
)

func TestTree(t *testing.T) {
	b := new(bytes.Buffer)
	tr := New("ostree/testdata")
	opts := &Options{
		Fs:      new(ostree.FS),
		OutFile: b,
	}
	tr.Visit(opts)
	tr.Print(opts)

	actual := b.String()

	expect := `ostree/testdata
├── a
│   └── b
│       └── b.txt
└── c
    └── c.txt
`
	if actual != expect {
		t.Errorf("\nactual\n%s\n != expect\n%s\n", actual, expect)
	}
}
