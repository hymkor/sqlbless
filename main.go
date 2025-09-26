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

// NewConfigFromFlag returns the constructor of Config from flag variables.
//
//	cfgSetup := NewConfigFromFlag()
//	flag.Parse()
//	cfg := cfgSetup()
func NewConfigFromFlag() func() *Config {
	var (
		flagCrLf           = flag.Bool("crlf", false, "use CRLF")
		flagFieldSeperator = flag.String("fs", ",", "Set field separator")
		flagNullString     = flag.String("null", "\u2400", "Set a string representing NULL")
		flagTsv            = flag.Bool("tsv", false, "Use TAB as seperator")
		flagSubmitByEnter  = flag.Bool("submit-enter", false, "Submit by [Enter] and insert a new line by [Ctrl]-[Enter]")
		flagScript         = flag.String("f", "", "script file")
		flagDebug          = flag.Bool("debug", false, "Print type in CSV")
		flagAuto           = flag.String("auto", "", "autopilot")
		flagTerm           = flag.String("term", ";", "SQL terminator to use instead of semicolon")
		flagSpool          = flag.String("spool", os.DevNull, "Spool filename")
	)
	flag.Usage = usage
	return func() *Config {
		return &Config{
			Auto:           *flagAuto,
			Term:           *flagTerm,
			CrLf:           *flagCrLf,
			Null:           *flagNullString,
			Tsv:            *flagTsv,
			FieldSeperator: *flagFieldSeperator,
			Debug:          *flagDebug,
			SubmitByEnter:  *flagSubmitByEnter,
			Script:         *flagScript,
			SpoolFilename:  *flagSpool,
		}
	}
}

func Main() error {
	writeSignature(os.Stdout)

	cfgSetup := NewConfigFromFlag()
	flag.Parse()
	cfg := cfgSetup()
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
