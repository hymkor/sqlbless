SQL-Bless
===========

The SQL-Bless is a command-line database client like SQL\*Plus or psql.

- Emacs-like keybindings for inline editing of multiple lines of SQL.
    - The action of `Enter` key will only insert a line feed code.
    - Press `Ctrl`-`Enter` to execute the input.
- Save the result of SELECT in CSV format
- Oracle and PostgreSQL are supported.
    - Any database supported by Go's "database/sql" can be used with a
small amount of extra code in `dbspecs.go`
- Auto commit is disabled.

![image](./demo.gif)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **Insert a linefeed** |
| `Ctrl`-`Enter`/`J` | **Execute text as SQL** |
| `Ctrl`-`F`/`B`/`N`/`P` | Editing like Emacs |
| `Ctrl`-`C` | Exit with rollback |
| `Ctrl`-`D` | Delete character or submit EOF (exit with rollback) |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | Insert the previous SQL (history)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | Insert the next SQL (history) |

Supported commands
------------------

- SELECT / INSERT / UPDATE / DELETE
    - INSERT, UPDATE and DELETE begin the transaction automatically.
- COMMIT / ROLLBACK
- SPOOL
    - `spool FILENAME` .. open FILENAME and write log and output.
    - `spool off` .. stop spooling and close.
- EXIT / QUIT
    - Rollback a transaction and exit SQL-Bless.
- START filename
    - Start the SQL script given with filename
- REM comments

- Semicolon `;` is a statement seperator when script is executed.
- When sql is input interactively, Semicolon `;` is ignored.

Example of a spooled file
--------------------------

``` CSV
# (2023-04-17 22:52:16)
# select *
#   from tab
#  where rownum < 5
TNAME,TABTYPE,CLUSTERID
AQ$_INTERNET_AGENTS,TABLE,<NULL>
AQ$_INTERNET_AGENT_PRIVS,TABLE,<NULL>
AQ$_KEY_SHARD_MAP,TABLE,<NULL>
AQ$_QUEUES,TABLE,<NULL>
# (2023-04-17 22:52:20)
# history
0,2023-04-17 22:52:05,spool hoge
1,2023-04-17 22:52:16,"select *
  from tab
 where rownum < 5"
2,2023-04-17 22:52:20,history
```

Install
-------

Download the binary package from [Releases](https://github.com/hymkor/sqlbless/releases) and extract the executable.

### for scoop-installer

```
scoop install https://raw.githubusercontent.com/hymkor/sqlbless/master/sqlbless.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install sqlbless
```

How to start
-------------

    $ sqlbless {options} DRIVERNAME "DATASOURCENAME"

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- Use https://github.com/sijms/go-ora

### PostgreSQL

    $ sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"

- Use https://github.com/lib/pq

### SQL Server

    $ sqlbless sqlserver "sqlserver://@localhost?database=master"
    ( Windows authentication )

- Use https://github.com/microsoft/go-mssqldb

### Common Options

- `-crlf`
    - Use CRLF
- `-fs string`
    - Set a field separator (default `","`)
- `-null string`
    - Set a string representing NULL (default `"<NULL>"`)
- `-tsv`
    - Use TAB as seperator
- `-f string`
    - Start the script
