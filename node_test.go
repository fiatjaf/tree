package main

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// Mock file/FileInfo
type file struct {
	name    string
	size    int64
	files   []*file
	lastMod time.Time
	stat    interface{}
	mode    os.FileMode
}

func (f file) Name() string { return f.name }
func (f file) Size() int64  { return f.size }
func (f file) Mode() (o os.FileMode) {
	if f.mode != o {
		return f.mode
	}
	if f.stat != nil {
		stat := (f.stat).(*syscall.Stat_t)
		o = os.FileMode(stat.Mode)
	}
	return
}
func (f file) ModTime() time.Time { return f.lastMod }
func (f file) IsDir() bool        { return nil != f.files }
func (f file) Sys() interface{} {
	if f.stat == nil {
		return new(syscall.Stat_t)
	}
	return f.stat
}

// Mock filesystem
type MockFs struct {
	files map[string]*file
}

func NewFs() *MockFs {
	return &MockFs{make(map[string]*file)}
}

func (fs *MockFs) clean() *MockFs {
	fs.files = make(map[string]*file)
	return fs
}

func (fs *MockFs) addFile(path string, file *file) *MockFs {
	fs.files[path] = file
	if file.IsDir() {
		for _, f := range file.files {
			fs.addFile(path+"/"+f.name, f)
		}
	}
	return fs
}

func (fs *MockFs) Stat(path string) (os.FileInfo, error) {
	if path == "root/bad" {
		return nil, errors.New("stat failed")
	}
	return fs.files[path], nil
}

func (fs *MockFs) ReadDir(path string) ([]string, error) {
	var names []string
	for _, file := range fs.files[path].files {
		names = append(names, file.Name())
	}
	return names, nil
}

// Mock output file
type Out struct {
	str string
}

func (o *Out) equal(s string) bool {
	return o.str == s
}

func (o *Out) Write(p []byte) (int, error) {
	o.str += string(p)
	return len(p), nil
}

func (o *Out) clear() {
	o.str = ""
}

// FileSystem and Stdout mocks
var (
	fs  = NewFs()
	out = new(Out)
)

type treeTest struct {
	name     string
	opts     *Options // test params.
	expected string   // expected output.
	dirs     int      // expected dir count.
	files    int      // expected file count.
}

var listTests = []treeTest{
	{"basic", &Options{Fs: fs, OutFile: out}, `root
├── a
├── b
├── c
│   ├── d
│   ├── e
│   ├── g
│   │   ├── h
│   │   └── i
│   └── k
└── j
`, 2, 8},
	{"all", &Options{Fs: fs, OutFile: out, All: true, NoSort: true}, `root
├── a
├── b
├── c
│   ├── d
│   ├── e
│   ├── .f
│   ├── g
│   │   ├── h
│   │   └── i
│   └── k
└── j
`, 2, 9},
	{"dirs", &Options{Fs: fs, OutFile: out, DirsOnly: true}, `root
└── c
    └── g
`, 2, 0},
	{"fullPath", &Options{Fs: fs, OutFile: out, FullPath: true}, `root
├── root/a
├── root/b
├── root/c
│   ├── root/c/d
│   ├── root/c/e
│   ├── root/c/g
│   │   ├── root/c/g/h
│   │   └── root/c/g/i
│   └── root/c/k
└── root/j
`, 2, 8},
	{"deepLevel", &Options{Fs: fs, OutFile: out, DeepLevel: 1}, `root
├── a
├── b
├── c
└── j
`, 1, 3},
	{"pattern (a|e|i)", &Options{Fs: fs, OutFile: out, Pattern: "(a|e|i)"}, `root
├── a
└── c
    ├── e
    └── g
        └── i
`, 2, 3},
	{"pattern (x) + 0 files", &Options{Fs: fs, OutFile: out, Pattern: "(x)"}, `root
└── c
    └── g
`, 2, 0},
	{"ipattern (a|e|i)", &Options{Fs: fs, OutFile: out, IPattern: "(a|e|i)"}, `root
├── b
├── c
│   ├── d
│   ├── g
│   │   └── h
│   └── k
└── j
`, 2, 5},
	{"pattern (A) + ignore-case", &Options{Fs: fs, OutFile: out, Pattern: "(A)", IgnoreCase: true}, `root
├── a
└── c
    └── g
`, 2, 1},
	{"pattern (A) + ignore-case + prune", &Options{Fs: fs, OutFile: out, Pattern: "(A)", Prune: true, IgnoreCase: true}, `root
└── a
`, 0, 1},
	{"pattern (a) + prune", &Options{Fs: fs, OutFile: out, Pattern: "(a)", Prune: true}, `root
└── a
`, 0, 1},
	{"pattern (c) + matchdirs", &Options{Fs: fs, OutFile: out, Pattern: "(c)", MatchDirs: true}, `root
└── c
    ├── d
    ├── e
    ├── g
    └── k
`, 2, 3},
	{"pattern (c.*) + matchdirs", &Options{Fs: fs, OutFile: out, Pattern: "(c.*)", MatchDirs: true}, `root
└── c
    ├── d
    ├── e
    ├── g
    │   ├── h
    │   └── i
    └── k
`, 2, 5},
	{"ipattern (c) + matchdirs", &Options{Fs: fs, OutFile: out, IPattern: "(c)", MatchDirs: true}, `root
├── a
├── b
└── j
`, 0, 3},
	{"ipattern (g) + matchdirs", &Options{Fs: fs, OutFile: out, IPattern: "(g)", MatchDirs: true}, `root
├── a
├── b
├── c
│   ├── d
│   ├── e
│   └── k
└── j
`, 1, 6},
	{"ipattern (a|e|i|h) + matchdirs + prune", &Options{Fs: fs, OutFile: out, IPattern: "(a|e|i|h)", MatchDirs: true, Prune: true}, `root
├── b
├── c
│   ├── d
│   └── k
└── j
`, 1, 4},
	{"pattern (d|e) + prune", &Options{Fs: fs, OutFile: out, Pattern: "(d|e)", Prune: true}, `root
└── c
    ├── d
    └── e
`, 1, 2},
	{"pattern (c.*) + matchdirs + prune ", &Options{Fs: fs, OutFile: out, Pattern: "(c.*)", Prune: true, MatchDirs: true}, `root
└── c
    ├── d
    ├── e
    ├── g
    │   ├── h
    │   └── i
    └── k
`, 2, 5},
}

func TestSimple(t *testing.T) {
	root := &file{
		name: "root",
		size: 200,
		files: []*file{
			{name: "a", size: 50},
			{name: "b", size: 50},
			{
				name: "c",
				size: 100,
				files: []*file{
					{name: "d", size: 50},
					{name: "e", size: 50},
					{name: ".f", size: 0},
					{
						name: "g",
						size: 100,
						files: []*file{
							{name: "h", size: 50},
							{name: "i", size: 50},
						},
					},
					{name: "k", size: 50},
				},
			},
			{name: "j", size: 50},
		},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range listTests {
		inf := New(root.name)
		d, f := inf.Visit(test.opts)
		if d != test.dirs {
			t.Errorf("wrong dir count for test %q:\ngot:\n%d\nexpected:\n%d", test.name, d, test.dirs)
		}
		if f != test.files {
			t.Errorf("wrong file count for test %q:\ngot:\n%d\nexpected:\n%d", test.name, f, test.files)
		}
		inf.Print(test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

var sortTests = []treeTest{
	{"name-sort", &Options{Fs: fs, OutFile: out, NameSort: true}, `root
├── a
├── b
└── c
    └── d
`, 1, 3},
	{"dirs-first sort", &Options{Fs: fs, OutFile: out, DirSort: true}, `root
├── c
│   └── d
├── b
└── a
`, 1, 3},
	{"reverse sort", &Options{Fs: fs, OutFile: out, ReverSort: true, DirSort: true}, `root
├── b
├── a
└── c
    └── d
`, 1, 3},
	{"no-sort", &Options{Fs: fs, OutFile: out, NoSort: true, DirSort: true}, `root
├── b
├── c
│   └── d
└── a
`, 1, 3},
	{"size-sort", &Options{Fs: fs, OutFile: out, SizeSort: true}, `root
├── a
├── c
│   └── d
└── b
`, 1, 3},
	{"last-mod-sort", &Options{Fs: fs, OutFile: out, ModSort: true}, `root
├── a
├── b
└── c
    └── d
`, 1, 3},
	{"c-time-sort", &Options{Fs: fs, OutFile: out, CTimeSort: true}, `root
├── b
├── c
│   └── d
└── a
`, 1, 3},
}

func TestSort(t *testing.T) {
	tFmt := "2006-Jan-02"
	aTime, _ := time.Parse(tFmt, "2015-Aug-01")
	bTime, _ := time.Parse(tFmt, "2015-Sep-01")
	cTime, _ := time.Parse(tFmt, "2015-Oct-01")
	root := &file{
		name: "root",
		size: 200,
		files: []*file{
			{name: "b", size: 11, lastMod: bTime},
			{name: "c", size: 10, files: []*file{{name: "d", size: 10, lastMod: cTime}}, lastMod: cTime},
			{name: "a", size: 9, lastMod: aTime},
		},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range sortTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print(test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

var graphicTests = []treeTest{
	{"no-indent", &Options{Fs: fs, OutFile: out, NoIndent: true}, `root
a
b
c
`, 0, 3},
	{"quotes", &Options{Fs: fs, OutFile: out, Quotes: true}, `"root"
├── "a"
├── "b"
└── "c"
`, 0, 3},
	{"byte-size", &Options{Fs: fs, OutFile: out, ByteSize: true}, `[      12499]  root
├── [       1500]  a
├── [       9999]  b
└── [       1000]  c
`, 0, 3},
	{"unit-size", &Options{Fs: fs, OutFile: out, UnitSize: true}, `[ 12K]  root
├── [1.5K]  a
├── [9.8K]  b
└── [1000]  c
`, 0, 3},
	{"show-gid", &Options{Fs: fs, OutFile: out, ShowGid: true}, `root
├── [1   ]  a
├── [2   ]  b
└── [1   ]  c
`, 0, 3},
	{"mode", &Options{Fs: fs, OutFile: out, FileMode: true}, `root
├── [-rw-r--r--]  a
├── [-rwxr-xr-x]  b
└── [-rw-rw-rw-]  c
`, 0, 3},
	{"lastMod", &Options{Fs: fs, OutFile: out, LastMod: true, Now: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}, `root
├── [Feb 11 00:00]  a
├── [Jan 28  2006]  b
└── [Jul 12 00:00]  c
`, 0, 3},
}

func TestGraphics(t *testing.T) {
	tFmt := "2006-Jan-02"
	aTime, _ := time.Parse(tFmt, "2015-Feb-11")
	bTime, _ := time.Parse(tFmt, "2006-Jan-28")
	cTime, _ := time.Parse(tFmt, "2015-Jul-12")
	root := &file{
		name: "root",
		size: 11499,
		files: []*file{
			{name: "a", size: 1500, lastMod: aTime, stat: &syscall.Stat_t{Gid: 1, Mode: 0644}},
			{name: "b", size: 9999, lastMod: bTime, stat: &syscall.Stat_t{Gid: 2, Mode: 0755}},
			{name: "c", size: 1000, lastMod: cTime, stat: &syscall.Stat_t{Gid: 1, Mode: 0666}},
		},
		stat: &syscall.Stat_t{Gid: 1},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range graphicTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print(test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

var symlinkTests = []treeTest{
	{"symlink", &Options{Fs: fs, OutFile: out}, `root
└── symlink -> root/symlink
`, 0, 1},
	{"symlink-rec", &Options{Fs: fs, OutFile: out, FollowLink: true}, `root
└── symlink -> root/symlink [recursive, not followed]
`, 0, 1},
}

func TestSymlink(t *testing.T) {
	root := &file{
		name: "root",
		files: []*file{
			{name: "symlink", mode: os.ModeSymlink, files: make([]*file, 0)},
		},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range symlinkTests {
		inf := New(root.name)
		inf.Visit(test.opts)
		inf.Print(test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}

func TestCount(t *testing.T) {
	defer out.clear()
	root := &file{
		name: "root",
		files: []*file{
			{
				name: "a",
				files: []*file{
					{
						name:  "b",
						files: []*file{{name: "c"}},
					},
					{
						name: "d",
						files: []*file{
							{
								name:  "e",
								files: []*file{{name: "f"}, {name: "g"}},
							},
						},
					},
					{
						name: "h",
						files: []*file{
							{
								name:  "i",
								files: []*file{{name: "j"}},
							},
							{
								name:  "k",
								files: []*file{{name: "l"}, {name: "m"}},
							},
							{name: "n"},
							{name: "o"},
						},
					},
				},
			},
		},
	}
	fs.clean().addFile(root.name, root)
	opt := &Options{Fs: fs, OutFile: out}
	inf := New(root.name)
	d, f := inf.Visit(opt)
	if d != 7 || f != 8 {
		inf.Print(opt)
		t.Errorf("TestCount - expect (dir, file) count to be equal to (7, 8)\n%s", out.str)
	}
}

var errorTests = []treeTest{
	{"basic", &Options{Fs: fs, OutFile: out}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"all", &Options{Fs: fs, OutFile: out, All: true, NoSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"dirs", &Options{Fs: fs, OutFile: out, DirsOnly: true}, `root
└── bad [stat failed]
`, 0, 0},
	{"fullPath", &Options{Fs: fs, OutFile: out, FullPath: true}, `root
├── root/a
├── root/b
├── root/j
└── root/bad [stat failed]
`, 0, 3},
	{"deepLevel", &Options{Fs: fs, OutFile: out, DeepLevel: 1}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"pattern (a|e|i)", &Options{Fs: fs, OutFile: out, Pattern: "(a|e|i)"}, `root
├── a
└── bad [stat failed]
`, 0, 1},
	{"pattern (x) + 0 files", &Options{Fs: fs, OutFile: out, Pattern: "(x)"}, `root
└── bad [stat failed]
`, 0, 0},
	{"ipattern (a|e|i)", &Options{Fs: fs, OutFile: out, IPattern: "(a|e|i)"}, `root
├── b
├── j
└── bad [stat failed]
`, 0, 2},
	{"pattern (A) + ignore-case", &Options{Fs: fs, OutFile: out, Pattern: "(A)", IgnoreCase: true}, `root
├── a
└── bad [stat failed]
`, 0, 1},
	{"pattern (A) + ignore-case + prune", &Options{Fs: fs, OutFile: out, Pattern: "(A)", Prune: true, IgnoreCase: true}, `root
├── a
└── bad [stat failed]
`, 0, 1},
	{"pattern (a) + prune", &Options{Fs: fs, OutFile: out, Pattern: "(a)", Prune: true}, `root
├── a
└── bad [stat failed]
`, 0, 1},
	{"pattern (c) + matchdirs", &Options{Fs: fs, OutFile: out, Pattern: "(c)", MatchDirs: true}, `root
└── bad [stat failed]
`, 0, 0},
	{"pattern (c.*) + matchdirs", &Options{Fs: fs, OutFile: out, Pattern: "(c.*)", MatchDirs: true}, `root
└── bad [stat failed]
`, 0, 0},
	{"ipattern (c) + matchdirs", &Options{Fs: fs, OutFile: out, IPattern: "(c)", MatchDirs: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"ipattern (g) + matchdirs", &Options{Fs: fs, OutFile: out, IPattern: "(g)", MatchDirs: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"ipattern (a|e|i|h) + matchdirs + prune", &Options{Fs: fs, OutFile: out, IPattern: "(a|e|i|h)", MatchDirs: true, Prune: true}, `root
├── b
├── j
└── bad [stat failed]
`, 0, 2},
	{"pattern (d|e) + prune", &Options{Fs: fs, OutFile: out, Pattern: "(d|e)", Prune: true}, `root
└── bad [stat failed]
`, 0, 0},
	{"pattern (c.*) + matchdirs + prune ", &Options{Fs: fs, OutFile: out, Pattern: "(c.*)", Prune: true, MatchDirs: true}, `root
└── bad [stat failed]
`, 0, 0},

	{"name-sort", &Options{Fs: fs, OutFile: out, NameSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"dirs-first sort", &Options{Fs: fs, OutFile: out, DirSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"reverse sort", &Options{Fs: fs, OutFile: out, ReverSort: true, NameSort: true}, `root
├── bad [stat failed]
├── j
├── b
└── a
`, 0, 3},
	{"no-sort", &Options{Fs: fs, OutFile: out, NoSort: true, DirSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"size-sort", &Options{Fs: fs, OutFile: out, SizeSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"last-mod-sort", &Options{Fs: fs, OutFile: out, ModSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
	{"c-time-sort", &Options{Fs: fs, OutFile: out, CTimeSort: true}, `root
├── a
├── b
├── j
└── bad [stat failed]
`, 0, 3},
}

func TestError(t *testing.T) {
	root := &file{
		name: "root",
		size: 200,
		files: []*file{
			{name: "a", size: 50},
			{name: "b", size: 50},
			{name: "j", size: 50},
			{name: "bad", size: 50}, // stat fails on this file
		},
	}
	fs.clean().addFile(root.name, root)
	for _, test := range errorTests {
		inf := New(root.name)
		d, f := inf.Visit(test.opts)
		if d != test.dirs {
			t.Errorf("wrong dir count for test %q:\ngot:\n%d\nexpected:\n%d", test.name, d, test.dirs)
		}
		if f != test.files {
			t.Errorf("wrong file count for test %q:\ngot:\n%d\nexpected:\n%d", test.name, f, test.files)
		}
		inf.Print(test.opts)
		if !out.equal(test.expected) {
			t.Errorf("%s:\ngot:\n%+v\nexpected:\n%+v", test.name, out.str, test.expected)
		}
		out.clear()
	}
}
