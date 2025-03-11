package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/a8m/tree"
	"github.com/a8m/tree/ostree"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:                   "tree",
		Usage:                  "List contents of directories in a tree-like format",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			// List options
			&cli.BoolFlag{Name: "a", Usage: "All files are listed"},
			&cli.BoolFlag{Name: "d", Usage: "List directories only"},
			&cli.BoolFlag{Name: "f", Usage: "Print the full path prefix for each file"},
			&cli.BoolFlag{Name: "ignore-case", Usage: "Ignore case when pattern matching"},
			&cli.BoolFlag{Name: "noreport", Usage: "Turn off file/directory count at end of tree listing"},
			&cli.BoolFlag{Name: "l", Usage: "Follow symbolic links like directories"},
			&cli.IntFlag{Name: "L", Value: 3, Usage: "Descend only level directories deep"},
			&cli.StringFlag{Name: "P", Usage: "List only those files that match the pattern given"},
			&cli.StringFlag{Name: "I", Usage: "Do not list files that match the given pattern"},
			&cli.StringFlag{Name: "o", Usage: "Output to file instead of stdout"},

			// Files options
			&cli.BoolFlag{Name: "1", Usage: "Print first line of text/plain files"},
			&cli.BoolFlag{Name: "s", Usage: "Print the size in bytes of each file"},
			&cli.BoolFlag{Name: "h", Usage: "Print the size in a more human readable way"},
			&cli.BoolFlag{Name: "p", Usage: "Print the protections for each file"},
			&cli.BoolFlag{Name: "u", Usage: "Displays file owner or UID number"},
			&cli.BoolFlag{Name: "g", Usage: "Displays file group owner or GID number"},
			&cli.BoolFlag{Name: "Q", Usage: "Quote filenames with double quotes"},
			&cli.BoolFlag{Name: "D", Usage: "Print the date of last modification or (-c) status change"},
			&cli.BoolFlag{Name: "inodes", Usage: "Print inode number of each file"},
			&cli.BoolFlag{Name: "device", Usage: "Print device ID number to which each file belongs"},

			// Sort options
			&cli.BoolFlag{Name: "U", Usage: "Leave files unsorted"},
			&cli.BoolFlag{Name: "v", Usage: "Sort files alphanumerically by version"},
			&cli.BoolFlag{Name: "t", Usage: "Sort files by last modification time"},
			&cli.BoolFlag{Name: "c", Usage: "Sort files by last status change time"},
			&cli.BoolFlag{Name: "r", Usage: "Reverse the order of the sort"},
			&cli.BoolFlag{Name: "dirsfirst", Usage: "List directories before files (-U disables)"},
			&cli.StringFlag{Name: "sort", Usage: "Select sort: name,version,size,mtime,ctime"},

			// Graphics options
			&cli.BoolFlag{Name: "i", Usage: "Don't print indentation lines"},
			&cli.BoolFlag{Name: "C", Usage: "Turn colorization on always"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var nd, nf int
			dirs := []string{"."}

			// Make it work with leading dirs
			if c.Args().Len() > 0 {
				dirs = c.Args().Slice()
			}

			// Output file
			outFile := os.Stdout
			var err error
			if c.String("o") != "" {
				outFile, err = os.Create(c.String("o"))
				if err != nil {
					return fmt.Errorf("tree: \"%s\"", err)
				}
				defer outFile.Close()
			}

			// Check sort-type
			if c.String("sort") != "" {
				switch c.String("sort") {
				case "version", "mtime", "ctime", "name", "size":
				default:
					msg := fmt.Sprintf("sort type '%s' not valid, should be one of: "+
						"name,version,size,mtime,ctime", c.String("sort"))
					return errors.New(msg)
				}
			}

			// Set options
			opts := &tree.Options{
				// Required
				Fs:      new(ostree.FS),
				OutFile: outFile,
				// List
				All:        c.Bool("a"),
				DirsOnly:   c.Bool("d"),
				FullPath:   c.Bool("f"),
				DeepLevel:  int(c.Int("L")),
				FollowLink: c.Bool("l"),
				Pattern:    c.String("P"),
				IPattern:   c.String("I"),
				IgnoreCase: c.Bool("ignore-case"),
				// Files
				Contents: c.Bool("1"),
				ByteSize: c.Bool("s"),
				UnitSize: c.Bool("h"),
				FileMode: c.Bool("p"),
				ShowUid:  c.Bool("u"),
				ShowGid:  c.Bool("g"),
				LastMod:  c.Bool("D"),
				Quotes:   c.Bool("Q"),
				Inodes:   c.Bool("inodes"),
				Device:   c.Bool("device"),
				// Sort
				NoSort:    c.Bool("U"),
				ReverSort: c.Bool("r"),
				DirSort:   c.Bool("dirsfirst"),
				VerSort:   c.Bool("v") || c.String("sort") == "version",
				ModSort:   c.Bool("t") || c.String("sort") == "mtime",
				CTimeSort: c.Bool("c") || c.String("sort") == "ctime",
				NameSort:  c.String("sort") == "name",
				SizeSort:  c.String("sort") == "size",
				// Graphics
				NoIndent: c.Bool("i"),
				Colorize: c.Bool("C"),
			}

			for _, dir := range dirs {
				inf := tree.New(dir)
				d, f := inf.Visit(opts)
				nd, nf = nd+d, nf+f
				inf.Print(opts)
			}

			// Print footer report
			if !c.Bool("noreport") {
				footer := fmt.Sprintf("\n%d directories", nd)
				if !opts.DirsOnly {
					footer += fmt.Sprintf(", %d files", nf)
				}
				fmt.Fprintln(outFile, footer)
			}

			return nil
		},
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:        "help",
		Usage:       "show help",
		HideDefault: true,
		Local:       true,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "tree: \"%s\"\n", err)
		os.Exit(1)
	}
}
