package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
)

func main() {
	opt := arg.New("demo", "A CLI demonstrating climate/arg")
	opt.SetFlag(arg.GroupDefault, "v", "verbose", "Enable verbose output")
	opt.SetOption(arg.GroupDefault, "n", "name", "Your name", "world", false, arg.VarString, nil)
	opt.SetOption(arg.GroupDefault, "c", "count", "Repeat count", 1, false, arg.VarInt, nil)
	opt.SetFlag(arg.GroupDefault, "", "shout", "Uppercase the greeting")
	opt.SetPositional("files", "Files to process", nil, false, arg.VarStringSlice)

	if err := opt.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	name := opt.GetString("name")
	count := opt.GetInt("count")
	files := opt.GetPosStringSlice("files")
	verbose := opt.GetBool("v")
	shout := opt.GetBool("shout")

	greeting := fmt.Sprintf("Hello, %s!", name)
	if shout {
		greeting = fmt.Sprintf("HELLO, %s!", name)
	}

	for i := 0; i < count; i++ {
		fmt.Println(greeting)
	}

	if verbose {
		fmt.Printf("Count: %d\n", count)
		fmt.Printf("Files: %v\n", files)
	}
}
