package sqlbless

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/hymkor/sqlbless/dbdialect"
)

func findDbDialect(args []string) (*dbdialect.DBDialect, []string, error) {
	spec, ok := dbdialect.DbDialects[strings.ToUpper(args[0])]
	if ok {
		if len(args) < 2 {
			return nil, nil, errors.New("DSN String is not specified")
		}
		return spec, []string{args[0], strings.Join(args[1:], " ")}, nil
	}
	scheme, _, ok := strings.Cut(args[0], ":")
	if ok {
		spec, ok = dbdialect.DbDialects[strings.ToUpper(scheme)]
		if ok {
			return spec, []string{scheme, strings.Join(args, " ")}, nil
		}
	}
	return nil, nil, fmt.Errorf("support driver not found: %s", args[0])
}

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
	for _, d := range dbdialect.DbDialects {
		fmt.Fprintf(w, "  %s\n", d.Usage)
	}
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
		flagNullString     = flag.String("null", "<NULL>", "Set a string representing NULL")
		flagTsv            = flag.Bool("tsv", false, "Use TAB as seperator")
		flagSubmitByEnter  = flag.Bool("submit-enter", false, "Submit by [Enter] and insert a new line by [Ctrl]-[Enter]")
		flagScript         = flag.String("f", "", "script file")
		flagDebug          = flag.Bool("debug", false, "Print type in CSV")
		flagAuto           = flag.String("auto", "", "autopilot")
		flagTerm           = flag.String("term", ";", "SQL terminator to use instead of semicolon")
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
	dbDialect, args, err := findDbDialect(args)
	if err != nil {
		return err
	}
	dataSourceName := args[1]
	if dbDialect.DSNFilter != nil {
		dataSourceName, err = dbDialect.DSNFilter(dataSourceName)
		if err != nil {
			return err
		}
		if cfg.Debug {
			fmt.Fprintln(os.Stderr, dataSourceName)
		}
	}

	return cfg.Run(args[0], dataSourceName, dbDialect)
}
