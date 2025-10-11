package sqlbless

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/hymkor/sqlbless/dialect"
)

var Version string

func writeSignature(w io.Writer) {
	fmt.Fprintf(w, "# SQL-Bless %s-%s-%s built with %s\n",
		Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func usage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage: %s {-options} [DRIVERNAME] DATASOURCENAME\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(w, "Example:")
	dialect.Each(
		func(_ string, d *dialect.Entry) bool {
			fmt.Fprintf(w, "  %s\n", d.Usage)
			return true
		},
	)
}

func Main() error {
	writeSignature(os.Stdout)

	cfg := NewConfig().Bind(flag.CommandLine)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		return nil
	}
	d, err := dialect.ReadDBInfoFromArgs(args)
	if err != nil {
		return err
	}
	if cfg.Debug {
		fmt.Fprintln(os.Stderr, d.DataSource)
	}
	return cfg.Run(d.Driver, d.DataSource, d.Dialect)
}
