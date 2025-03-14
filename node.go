package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// Node represent some node in the tree
// contains FileInfo, and its childs
type Node struct {
	os.FileInfo
	path   string
	depth  int
	err    error
	nodes  Nodes
	vpaths map[string]bool
}

// List of nodes
type Nodes []*Node

// To use this package programmatically, you must implement this
// interface.
// For example: PTAL on 'cmd/tree/tree.go'
type Fs interface {
	Stat(path string) (os.FileInfo, error)
	ReadDir(path string) ([]string, error)
}

// Options store the configuration for specific tree.
// Note, that 'Fs', and 'OutFile' are required (OutFile can be os.Stdout).
type Options struct {
	Fs      Fs
	OutFile io.Writer
	// List
	All        bool
	DirsOnly   bool
	FullPath   bool
	IgnoreCase bool
	FollowLink bool
	DeepLevel  int
	Pattern    string
	IPattern   string
	MatchDirs  bool
	Prune      bool
	// File
	Contents bool
	ByteSize bool
	UnitSize bool
	FileMode bool
	ShowUid  bool
	ShowGid  bool
	LastMod  bool
	Quotes   bool
	Inodes   bool
	Device   bool
	// Sort
	NoSort    bool
	VerSort   bool
	ModSort   bool
	DirSort   bool
	NameSort  bool
	SizeSort  bool
	CTimeSort bool
	ReverSort bool
	// Graphics
	NoIndent bool
	Colorize bool
	// Color defaults to ANSIColor()
	Color func(*Node, string) string
	Now   time.Time
}

func (opts *Options) color(node *Node, s string) string {
	f := opts.Color
	if f == nil {
		f = ANSIColor
	}
	return f(node, s)
}

// New get path and create new node(root).
func New(path string) *Node {
	return &Node{path: path, vpaths: make(map[string]bool)}
}

// Visit all files under the given node.
func (node *Node) Visit(opts *Options) (dirs, files int) {
	// visited paths
	if path, err := filepath.Abs(node.path); err == nil {
		path = filepath.Clean(path)
		node.vpaths[path] = true
	}
	// stat
	fi, err := opts.Fs.Stat(node.path)
	if err != nil {
		node.err = err
		return
	}
	node.FileInfo = fi
	if !fi.IsDir() {
		return 0, 1
	}
	// increase dirs only if it's a dir, but not the root.
	if node.depth != 0 {
		dirs++
	}
	// DeepLevel option
	if opts.DeepLevel > 0 && opts.DeepLevel <= node.depth {
		return
	}
	// MatchDirs option
	var dirMatch bool
	if node.depth != 0 && opts.MatchDirs {
		// then disable prune and pattern for immediate children
		if opts.Pattern != "" {
			dirMatch = node.match(opts.Pattern, opts)
		} else if opts.IPattern != "" && node.match(opts.IPattern, opts) {
			return
		}
	}
	names, err := opts.Fs.ReadDir(node.path)
	if err != nil {
		node.err = err
		return
	}
	node.nodes = make(Nodes, 0)
	for _, name := range names {
		// "all" option
		if !opts.All && strings.HasPrefix(name, ".") {
			continue
		}
		nnode := &Node{
			path:   filepath.Join(node.path, name),
			depth:  node.depth + 1,
			vpaths: node.vpaths,
		}
		d, f := nnode.Visit(opts)
		if nnode.err == nil {
			if nnode.IsDir() {
				// "prune" option, hide empty directories
				if opts.Prune && f == 0 {
					continue
				}
				if opts.MatchDirs && opts.IPattern != "" && nnode.match(opts.IPattern, opts) {
					continue
				}
			} else {
				// "dirs only" option
				if opts.DirsOnly {
					continue
				}
				// Pattern matching
				if !dirMatch && opts.Pattern != "" && !nnode.match(opts.Pattern, opts) {
					continue
				}
				// IPattern matching
				if opts.IPattern != "" && nnode.match(opts.IPattern, opts) {
					continue
				}
			}
		}
		node.nodes = append(node.nodes, nnode)
		dirs, files = dirs+d, files+f
	}
	// Sorting
	if !opts.NoSort {
		node.sort(opts)
	}
	return
}

func (node *Node) match(pattern string, opt *Options) bool {
	var prefix string
	if opt.IgnoreCase {
		prefix = "(?i)"
	}
	search := node.Name()
	if strings.Contains(pattern, "*") {
		search = node.path
	}
	re, err := regexp.Compile(prefix + pattern)
	return err == nil && re.FindString(search) != ""
}

func (node *Node) sort(opts *Options) {
	var fn SortFunc
	switch {
	case opts.ModSort:
		fn = ModSort
	case opts.CTimeSort:
		fn = CTimeSort
	case opts.DirSort:
		fn = DirSort
	case opts.VerSort:
		fn = VerSort
	case opts.SizeSort:
		fn = SizeSort
	case opts.NameSort:
		fn = NameSort
	default:
		fn = NameSort // Default should be sorted, not unsorted.
	}
	if fn != nil {
		if opts.ReverSort {
			sort.Sort(sort.Reverse(ByFunc{node.nodes, fn}))
		} else {
			sort.Sort(ByFunc{node.nodes, fn})
		}
	}
}

// Path returns the Node's absolute path
func (node *Node) Path() string {
	return node.path
}

// Print nodes based on the given configuration.
func (node *Node) Print(opts *Options) { node.print("", opts) }

func dirRecursiveSize(opts *Options, node *Node) (size int64, err error) {
	if opts.DeepLevel > 0 && node.depth >= opts.DeepLevel {
		err = errors.New("Depth too high")
	}

	for _, nnode := range node.nodes {
		if nnode.err != nil {
			err = nnode.err
			continue
		}

		if !nnode.IsDir() {
			size += nnode.Size()
		} else {
			nsize, e := dirRecursiveSize(opts, nnode)
			size += nsize
			if e != nil {
				err = e
			}
		}
	}
	return
}

var reusable = make([]byte, 60)

func (node *Node) print(indent string, opts *Options) {
	if node.err != nil {
		err := node.err.Error()
		if msgs := strings.Split(err, ": "); len(msgs) > 1 {
			err = msgs[1]
		}
		name := node.path
		if !opts.FullPath {
			name = filepath.Base(name)
		}
		fmt.Fprintf(opts.OutFile, "%s [%s]\n", name, err)
		return
	}
	if !node.IsDir() {
		var props []string
		ok, inode, device, uid, gid := getStat(node)
		// inodes
		if ok && opts.Inodes {
			props = append(props, fmt.Sprintf("%d", inode))
		}
		// device
		if ok && opts.Device {
			props = append(props, fmt.Sprintf("%3d", device))
		}
		// Mode
		if opts.FileMode {
			props = append(props, node.Mode().String())
		}
		// Owner/Uid
		if ok && opts.ShowUid {
			uidStr := strconv.Itoa(int(uid))
			if u, err := user.LookupId(uidStr); err != nil {
				props = append(props, fmt.Sprintf("%-8s", uidStr))
			} else {
				props = append(props, fmt.Sprintf("%-8s", u.Username))
			}
		}
		// Gorup/Gid
		// TODO: support groupname
		if ok && opts.ShowGid {
			gidStr := strconv.Itoa(int(gid))
			props = append(props, fmt.Sprintf("%-4s", gidStr))
		}
		// Size
		if opts.ByteSize || opts.UnitSize {
			var size string
			if opts.UnitSize {
				size = fmt.Sprintf("%4s", formatBytes(node.Size()))
			} else {
				size = fmt.Sprintf("%11d", node.Size())
			}
			props = append(props, size)
		}
		// Last modification
		if opts.LastMod {
			t := opts.Now
			if t.IsZero() {
				t = time.Now()
			}

			format := "Jan 02 15:04"
			if node.ModTime().Year() != t.Year() {
				format = "Jan 02  2006"
			}

			props = append(props, node.ModTime().Format(format))
		}
		// Print properties
		if len(props) > 0 {
			fmt.Fprintf(opts.OutFile, "[%s]  ", strings.Join(props, " "))
		}
	} else {
		var props []string
		// Size
		if opts.ByteSize || opts.UnitSize {
			var size string
			rsize, err := dirRecursiveSize(opts, node)
			if err != nil && rsize <= 0 {
				if opts.UnitSize {
					size = "    "
				} else {
					size = "           "
				}
			} else if opts.UnitSize {
				size = fmt.Sprintf("%4s", formatBytes(rsize))
			} else {
				size = fmt.Sprintf("%11d", rsize)
			}
			props = append(props, size)
		}
		// Print properties
		if len(props) > 0 {
			fmt.Fprintf(opts.OutFile, "[%s]  ", strings.Join(props, " "))
		}
	}
	// name/path
	var name string
	if node.depth == 0 || opts.FullPath {
		name = node.path
	} else {
		name = node.Name()
	}
	// Quotes
	if opts.Quotes {
		name = fmt.Sprintf("\"%s\"", name)
	}
	// Colorize
	if opts.Colorize {
		name = opts.color(node, name)
	}
	// IsSymlink
	if node.Mode()&os.ModeSymlink == os.ModeSymlink {
		vtarget, err := os.Readlink(node.path)
		if err != nil {
			vtarget = node.path
		}
		targetPath, err := filepath.EvalSymlinks(node.path)
		if err != nil {
			targetPath = vtarget
		}
		fi, err := opts.Fs.Stat(targetPath)
		if opts.Colorize && fi != nil {
			vtarget = opts.color(&Node{FileInfo: fi, path: vtarget}, vtarget)
		}
		name = fmt.Sprintf("%s -> %s", name, vtarget)
		// Follow symbolic links like directories
		if opts.FollowLink {
			path, err := filepath.Abs(targetPath)
			if err == nil && fi != nil && fi.IsDir() {
				if _, ok := node.vpaths[filepath.Clean(path)]; !ok {
					inf := &Node{FileInfo: fi, path: targetPath}
					inf.vpaths = node.vpaths
					inf.Visit(opts)
					node.nodes = inf.nodes
				} else {
					name += " [recursive, not followed]"
				}
			}
		}
	}
	// Print file name/details
	// the main idea of the print logic came from here: github.com/campoy/tools/tree
	fmt.Fprint(opts.OutFile, name)

	// Print first line of content
	if opts.Contents {
		mime, _ := exec.Command("file", "--mime-type", "--brief", "-P", "bytes=200", node.path).Output()
		if unsafe.String(unsafe.SliceData(mime), len(mime)-1) == "text/plain" {
			if file, err := os.Open(node.path); err == nil {
				if n, err := file.Read(reusable); err == nil {
					if firstNewline := bytes.IndexAny(reusable[0:n], "\n\r"); firstNewline != -1 {
						n = firstNewline
					}
					hasMore := n == 60
					if hasMore {
						n = 59
					}

					fmt.Fprintf(opts.OutFile, " => `")
					opts.OutFile.Write(reusable[0:n])

					if hasMore {
						fmt.Fprintf(opts.OutFile, "…`")
					} else {
						fmt.Fprintf(opts.OutFile, "`")
					}
				}
				file.Close()
			}
		}
	}
	fmt.Fprintln(opts.OutFile, "")

	// tree stuff
	add := "│   "
	for i, nnode := range node.nodes {
		if opts.NoIndent {
			add = ""
		} else {
			if i == len(node.nodes)-1 {
				fmt.Fprintf(opts.OutFile, indent+"└── ")
				add = "    "
			} else {
				fmt.Fprintf(opts.OutFile, indent+"├── ")
			}
		}
		nnode.print(indent+add, opts)
	}
}

const (
	_        = iota // ignore first value by assigning to blank identifier
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
)

// Convert bytes to human readable string. Like a 2 MB, 64.2 KB, 52 B
func formatBytes(i int64) (result string) {
	var n float64
	sFmt, eFmt := "%.01f", ""
	switch {
	case i > EB:
		eFmt = "E"
		n = float64(i) / float64(EB)
	case i > PB:
		eFmt = "P"
		n = float64(i) / float64(PB)
	case i > TB:
		eFmt = "T"
		n = float64(i) / float64(TB)
	case i > GB:
		eFmt = "G"
		n = float64(i) / float64(GB)
	case i > MB:
		eFmt = "M"
		n = float64(i) / float64(MB)
	case i > KB:
		eFmt = "K"
		n = float64(i) / float64(KB)
	default:
		sFmt = "%.0f"
		n = float64(i)
	}
	if eFmt != "" && n >= 10 {
		sFmt = "%.0f"
	}
	result = fmt.Sprintf(sFmt+eFmt, n)
	result = strings.Trim(result, " ")
	return
}
