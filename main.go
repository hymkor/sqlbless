package sqlbless

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"unicode/utf8"

	"github.com/hymkor/struct2flag"

	"github.com/hymkor/sqlbless/dialect"
)

type Config struct {
	Auto           string `flag:"auto,autopilot"`
	Term           string `flag:"term,SQL terminator to use instead of semicolon"`
	CrLf           bool   `flag:"crlf,Use CRLF"`
	Null           string `flag:"null,Set a string representing NULL"`
	Tsv            bool   `flag:"tsv,Use TAB as seperator"`
	FieldSeperator string `flag:"fs,Set field separator"`
	Debug          bool   `flag:"debug,Print type in CSV"`
	SubmitByEnter  bool   `flag:"submit-enter,Submit by [Enter] and insert a new line by [Ctrl]-[Enter]"`
	Script         string `flag:"f,script file"`
	SpoolFilename  string `flag:"spool,Spool filename"`
	ReverseVideo   bool   `flag:"rv,Enable reverse-video display (invert foreground and background colors)"`
	DebugBell      bool   `flag:"debug-bell,Enable Debug Bell"`
}

func (cfg *Config) comma() byte {
	if cfg.Tsv {
		return '\t'
	}
	if len(cfg.FieldSeperator) > 0 {
		c, _ := utf8.DecodeRuneInString(cfg.FieldSeperator)
		return byte(c)
	}
	return ','
}

func New() *Config {
	return &Config{
		FieldSeperator: ",",
		Null:           "\u2400",
		Term:           ";",
		SpoolFilename:  os.DevNull,
	}
}

func (cfg *Config) Bind(fs *flag.FlagSet) *Config {
	struct2flag.Bind(fs, cfg)
	return cfg
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
	dialect.Each(
		func(_ string, d *dialect.Entry) bool {
			fmt.Fprintf(w, "  %s\n", d.Usage)
			return true
		},
	)
}

func Run() error {
	writeSignature(os.Stdout)

	cfg := New().Bind(flag.CommandLine)
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
