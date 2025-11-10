package sqlbless

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hymkor/sqlbless/dialect"
)

func TestSavePoint(t *testing.T) {
	restoreColor := disableColor()
	defer restoreColor()

	tmpDir := t.TempDir()
	testLst := filepath.Join(tmpDir, "output.lst")
	scriptPath := filepath.Join(tmpDir, "script.sql")
	w, err := os.Create(scriptPath)
	if err != nil {
		t.Fatal(err.Error())
	}

	fmt.Fprintf(w, `
		CREATE TABLE TESTTBL
		( SERIAL  NUMERIC,
		  STR     CHAR VARYING(20),
		  DT      CHARACTER VARYING(20),
		 PRIMARY  KEY (SERIAL) );

		INSERT INTO TESTTBL VALUES
		(10,'HOGE','2024-05-25 13:45:33');

		SAVEPOINT SP1;

		INSERT INTO TESTTBL VALUES
		(20,'HOGE','2024-05-25 13:45:33');

		ROLLBACK TO SP1;

		SPOOL %s;

		SELECT "CNT=",COUNT(*) FROM TESTTBL WHERE STR = 'HOGE';

		SPOOL OFF;

		ROLLBACK;

		EXIT;
`, testLst)
	w.Close()

	cfg := New()
	d, err := dialect.ReadDBInfoFromArgs([]string{"sqlite3", ":memory:"})
	if err != nil {
		t.Fatal(err.Error())
	}
	cfg.Script = scriptPath
	err = cfg.Run(d.Driver, d.DataSource, d.Dialect)
	if err != nil {
		t.Fatal(err.Error())
	}

	fd, err := os.Open(testLst)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer fd.Close()

	csvr := csv.NewReader(fd)
	csvr.Comment = '#'
	csvr.FieldsPerRecord = -1
	for {
		record, err := csvr.Read()
		if err == io.EOF {
			t.Fatal("target record not found")
		}
		//println(strings.Join(record, "|"))
		if record[0] == "CNT=" && record[1] == "1" {
			return
		}
	}
}
