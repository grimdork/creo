package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
)

func main() {
	opt := arg.New("tool", "A CLI with subcommands")

	opt.SetCommand("serve", "Run the server", arg.GroupDefault, func(sub *arg.Options) error {
		sub.SetOption(arg.GroupDefault, "p", "port", "Listen port", 8080, false, arg.VarInt, nil)
		sub.SetFlag(arg.GroupDefault, "", "tls", "Enable TLS")
		sub.SetOption(arg.GroupDefault, "", "host", "Bind address", "127.0.0.1", false, arg.VarString, nil)
		if err := sub.Parse(os.Args[2:]); err != nil {
			return err
		}
		port := sub.GetInt("port")
		host := sub.GetString("host")
		tls := sub.GetBool("tls")
		proto := "http"
		if tls {
			proto = "https"
		}
		fmt.Printf("Server listening on %s://%s:%d\n", proto, host, port)
		return nil
	}, []string{"s"})

	opt.SetCommand("config", "Show or set configuration", arg.GroupDefault, func(sub *arg.Options) error {
		sub.SetFlag(arg.GroupDefault, "", "show", "Display current configuration")
		sub.SetOption(arg.GroupDefault, "", "set", "Set key=value", "", false, arg.VarString, nil)
		if err := sub.Parse(os.Args[2:]); err != nil {
			return err
		}
		if sub.GetBool("show") {
			fmt.Println("Config: port=8080, host=127.0.0.1")
		}
		if kv := sub.GetString("set"); kv != "" {
			fmt.Printf("Set: %s\n", kv)
		}
		return nil
	}, []string{"cfg", "c"})

	opt.SetCommand("version", "Print version", arg.GroupDefault, func(sub *arg.Options) error {
		fmt.Println("tool v1.0.0")
		return nil
	}, []string{"v", "ver"})

	if err := opt.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
