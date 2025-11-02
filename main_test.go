package sqlbless

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hymkor/sqlbless/dialect"
	_ "github.com/hymkor/sqlbless/dialect/sqlite"
)

func disableColor() (restore func()) {
	if noColor, ok := os.LookupEnv("NO_COLOR"); ok {
		restore = func() { os.Setenv("NO_COLOR", noColor) }
	} else {
		restore = func() { os.Unsetenv("NO_COLOR") }
	}
	os.Setenv("NO_COLOR", "1")
	return
}

func TestConfigRun(t *testing.T) {
	restoreColor := disableColor()
	defer restoreColor()

	testLst := filepath.Join(t.TempDir(), "output.lst")
	auto :=
		"CREATE TABLE TESTTBL|" +
			"( TESTNO  NUMERIC ,|" +
			"  DT      CHARACTER VARYING(20) ,|" +
			" PRIMARY  KEY (TESTNO) )||" +
			"INSERT INTO TESTTBL VALUES|" +
			"(10,'2024-05-25 13:45:33')||" +
			"COMMIT||" +
			"EDIT TESTTBL||" +
			"/10|lr2015-06-07 20:21:22|cyy" +
			"SPOOL " + testLst + "||" +
			"SELECT * FROM TESTTBL||" +
			"SPOOL OFF||" +
			"ROLLBACK||" +
			"EXIT||"

	cfg := New()
	cfg.Auto = auto
	cfg.Debug = true
	d, err := dialect.ReadDBInfoFromArgs([]string{"sqlite3", ":memory:"})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = cfg.Run(d.Driver, d.DataSource, d.Dialect)
	if err != nil {
		t.Fatal(err.Error())
	}
	fd, err := os.Open(testLst)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer fd.Close()
	sc := bufio.NewScanner(fd)
	count := 0
	for sc.Scan() {
		line := sc.Text()
		if len(line) > 0 && line[0] == '#' {
			continue
		}
		count++
		if count == 1 {
			continue
		}
		field := strings.Split(line, ",")
		if len(field) >= 2 && field[1] == "2015-06-07 20:21:22" {
			// OK
			return
		}
	}
	if sc.Err() != nil {
		t.Fatal(err.Error())
	}
	t.Fatalf("%s: not found", "2015-06-07 20:21:22")
}
