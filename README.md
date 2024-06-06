SQL-Bless
===========

**&lt;English&gt;** / [&lt;Japanese&gt;](./README_ja.md)

The SQL-Bless is a command-line database client like SQL\*Plus or psql.

- Emacs-like keybindings for inline editing of multiple lines of SQL.
    - The action of `Enter` key will only insert a line feed code.
    - Press `Ctrl-J` or `Ctrl`-`Enter` to execute the input.
- Save the result of SELECT in CSV format
- Supported RDBMS
    - SQLite3
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- Allows editing database records directly, similar to a spreadsheet (with the `EDIT` command)
- Auto commit is disabled.

![image](./demo.gif)

[Video](https://www.youtube.com/watch?v=_cxBQKpfUds) by [@emisjerry](https://github.com/emisjerry)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **Insert a linefeed** |
| `Ctrl`-`Enter`/`J` | **Execute text as SQL** |
| `Ctrl`-`F`/`B` | Move Cursor forward or backward |
| `Ctrl`-`N`/`P` | Move Cursor or refer history |
| `Ctrl`-`C` | Exit with rollback |
| `Ctrl`-`D` | Delete character or submit EOF (exit with rollback) |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | Insert the previous SQL (history)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | Insert the next SQL (history) |

Supported commands
------------------

- `SELECT` / `INSERT` / `UPDATE` / `DELETE`
    - `INSERT`, `UPDATE` and `DELETE` begin the transaction automatically.
- `COMMIT` / `ROLLBACK`
- `SPOOL`
    - `spool FILENAME` .. open FILENAME and write log and output.
    - `spool off` .. stop spooling and close.
- `EXIT` / `QUIT`
    - Rollback a transaction and exit SQL-Bless.
- `START filename`
    - Start the SQL script given with filename
- `REM comments`
- `DESC [tablename]` / `\D [tablename]`
    - When the tablename is given, show the specification of the the table
    - Without the tablename, show the list of tables.
- `HISTORY`
    - Show the history of input SQLs
- `EDIT tablename [WHERE conditions...]`
    - Start an [editor][csvi] to modify the selected records of the table.
    - In the editor, these keys are bound.
        - `x` or `d`: set NULL to the current cell
        - `c`: apply changes
        - `q` or `ESC`: quit without applying changes
    - Because the EDIT statement automatically generates SQL from data changed in the editor, it may not be able to properly represent SQL data for special types specific to individual databases. If you find it, we would appreciate it if you could [contact us](https://github.com/hymkor/sqlbless/issues/new).

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

### via Scoop-installer (Windows)

```
scoop install https://raw.githubusercontent.com/hymkor/sqlbless/master/sqlbless.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install sqlbless
```

### Installing via Go

```
go install github.com/hymkor/sqlbless@latest
```

+ CGO is required on Windows-386 architecture.

|       | Windows      | Linux
|-------|--------------|--------
| 386   | CGO required | PureGo
| amd64 | PureGo       | PureGo

How to start
-------------

    $ sqlbless {options} DRIVERNAME "DATASOURCENAME"

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- The drivers used are
    - https://github.com/mattn/go-sqlite3 (Windows-386)
    - https://github.com/glebarez/go-sqlite (Linux and Windows-amd64)

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- The driver used is https://github.com/sijms/go-ora

### PostgreSQL

    $ sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"
    $ sqlbless postgres "postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full"

- The driver used is https://github.com/lib/pq

### SQL Server

    $ sqlbless sqlserver "sqlserver://@localhost?database=master"
    ( Windows authentication )

- The driver used is https://github.com/microsoft/go-mssqldb

### MySQL

    $ sqlbless.exe mysql "user:password@/database?parseTime=true&loc=Asia%2FTokyo"

- The driver used is http://github.com/go-sql-driver/mysql
- When both parseTime and loc are not specified, the value of TIMESTAMP is not expressed correctly

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
- `-submit-enter`
    - Submit by [Enter] and insert a new line by [Ctrl]-[Enter]
- `-debug`
    - Print type-information in the header of `SELECT` and `EDIT`
- `-help`
    - Help

[csvi]: https://github.com/hymkor/csvi
