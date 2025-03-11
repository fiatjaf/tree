package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/fiatjaf/tree/ostree"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:                   "tree",
		Usage:                  "List contents of directories in a tree-like format",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			// List options
			&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "All files are listed"},
			&cli.BoolFlag{Name: "dir", Aliases: []string{"d"}, Usage: "List directories only"},
			&cli.BoolFlag{Name: "full", Aliases: []string{"f"}, Usage: "Print the full path prefix for each file"},
			&cli.BoolFlag{Name: "ignore-case", Usage: "Ignore case when pattern matching"},
			&cli.BoolFlag{Name: "noreport", Usage: "Turn off file/directory count at end of tree listing"},
			&cli.BoolFlag{Name: "follow", Aliases: []string{"l"}, Usage: "Follow symbolic links like directories"},
			&cli.IntFlag{Name: "level", Aliases: []string{"max-depth", "L"}, Value: 3, Usage: "Descend only level directories deep"},
			&cli.StringFlag{Name: "pattern", Aliases: []string{"P"}, Usage: "List only those files that match the pattern given"},
			&cli.StringFlag{Name: "ignore", Aliases: []string{"I"}, Usage: "Do not list files that match the given pattern"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output to file instead of stdout"},

			// Files options
			&cli.BoolFlag{Name: "firstline", Aliases: []string{"1"}, Usage: "Print first line of text/plain files"},
			&cli.BoolFlag{Name: "size", Aliases: []string{"s"}, Usage: "Print the size in bytes of each file"},
			&cli.BoolFlag{Name: "human", Aliases: []string{"h"}, Usage: "Print the size in a more human readable way"},
			&cli.BoolFlag{Name: "protections", Aliases: []string{"p"}, Usage: "Print the protections for each file"},
			&cli.BoolFlag{Name: "owner", Aliases: []string{"u"}, Usage: "Displays file owner or UID number"},
			&cli.BoolFlag{Name: "group", Aliases: []string{"g"}, Usage: "Displays file group owner or GID number"},
			&cli.BoolFlag{Name: "quote", Aliases: []string{"Q"}, Usage: "Quote filenames with double quotes"},
			&cli.BoolFlag{Name: "date", Aliases: []string{"D"}, Usage: "Print the date of last modification or (-c) status change"},
			&cli.BoolFlag{Name: "inodes", Usage: "Print inode number of each file"},
			&cli.BoolFlag{Name: "device", Usage: "Print device ID number to which each file belongs"},

			// Sort options
			&cli.BoolFlag{Name: "unsorted", Aliases: []string{"U"}, Usage: "Leave files unsorted"},
			&cli.BoolFlag{Name: "v", Usage: "Sort files alphanumerically by version"},
			&cli.BoolFlag{Name: "t", Usage: "Sort files by last modification time"},
			&cli.BoolFlag{Name: "c", Usage: "Sort files by last status change time"},
			&cli.BoolFlag{Name: "reverse", Aliases: []string{"r"}, Usage: "Reverse the order of the sort"},
			&cli.BoolFlag{Name: "dirsfirst", Usage: "List directories before files (-U disables)"},
			&cli.StringFlag{Name: "sort", Usage: "Select sort: name,version,size,mtime,ctime"},

			// Graphics options
			&cli.BoolFlag{Name: "no-indent", Aliases: []string{"i"}, Usage: "Don't print indentation lines"},
			&cli.BoolFlag{Name: "colorize", Aliases: []string{"C"}, Usage: "Turn colorization on always"},
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
			if c.String("output") != "" {
				outFile, err = os.Create(c.String("output"))
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
			opts := &Options{
				// Required
				Fs:      new(ostree.FS),
				OutFile: outFile,
				// List
				All:        c.Bool("all"),
				DirsOnly:   c.Bool("dir"),
				FullPath:   c.Bool("full"),
				DeepLevel:  int(c.Int("level")),
				FollowLink: c.Bool("follow"),
				Pattern:    c.String("pattern"),
				IPattern:   c.String("ignore"),
				IgnoreCase: c.Bool("ignore-case"),
				// Files
				Contents: c.Bool("firstline"),
				ByteSize: c.Bool("size"),
				UnitSize: c.Bool("human"),
				FileMode: c.Bool("protections"),
				ShowUid:  c.Bool("owner"),
				ShowGid:  c.Bool("group"),
				LastMod:  c.Bool("date"),
				Quotes:   c.Bool("quote"),
				Inodes:   c.Bool("inodes"),
				Device:   c.Bool("device"),
				// Sort
				NoSort:    c.Bool("unsorted"),
				ReverSort: c.Bool("reverse"),
				DirSort:   c.Bool("dirs-first"),
				VerSort:   c.Bool("v") || c.String("sort") == "version",
				ModSort:   c.Bool("t") || c.String("sort") == "mtime",
				CTimeSort: c.Bool("c") || c.String("sort") == "ctime",
				NameSort:  c.String("sort") == "name",
				SizeSort:  c.String("sort") == "size",
				// Graphics
				NoIndent: c.Bool("no-indent"),
				Colorize: c.Bool("colorize"),
			}

			for _, dir := range dirs {
				inf := New(dir)
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

	cli.VersionFlag = &cli.BoolFlag{
		Name:        "version",
		Usage:       "print the version",
		HideDefault: true,
		Local:       true,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "tree: \"%s\"\n", err)
		os.Exit(1)
	}
}
