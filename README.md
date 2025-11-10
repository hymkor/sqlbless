SQL-Bless
===========

<!-- badges.cmd | -->
[![Go Test](https://github.com/hymkor/sqlbless/actions/workflows/go.yml/badge.svg)](https://github.com/hymkor/sqlbless/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-MIT-red)](https://github.com/hymkor/sqlbless/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hymkor/sqlbless.svg)](https://pkg.go.dev/github.com/hymkor/sqlbless)
<!-- -->

**&lt;English&gt;** / [&lt;Japanese&gt;](./README_ja.md)

SQL-Bless is a command-line database client similar to SQL\*Plus, supporting multiple database systems.

It was designed with reliability and transparency in mind.
In the past, I had to deliver an unfinished system due to client pressure,
which led to serious data inconsistencies that took enormous effort to fix.
That experience made me realize how important it is to have a tool
that prevents silent data corruption and keeps a complete, traceable log of every operation.
SQL-Bless is the result of that lesson — a tool that values safety over convenience.

- Emacs-like keybindings for editing multi-line SQL input.
    - Pressing Enter inserts a line break by default.
    - Use the ↑ (Up) arrow key or Ctrl-P to move the cursor to the previous line for editing.
    - Press Ctrl-J or Ctrl-Enter to execute the input immediately.
    - When you press Enter alone, the input is also executed if the last line ends with a semicolon or if the first word is a non-SQL command such as `EXIT` or `QUIT`.
- Save the result of SELECT in CSV format
- Supported RDBMS
    - SQLite3
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- Allows editing database records directly, similar to a spreadsheet (with the `EDIT` command)
- Transaction mode (auto-commit disabled)
    - A transaction is automatically started when executing DML (INSERT/UPDATE/DELETE)
    - Users cannot input a BEGIN statement (transactions are managed automatically)
    - Transactions can be ended with COMMIT or ROLLBACK
    - Whether DDL (CREATE/ALTER/DROP etc.) can be executed inside a transaction depends on the database:
        - **PostgreSQL**: All DDL can be executed inside a transaction except VACUUM, REINDEX, CLUSTER, CREATE/DROP DATABASE, and CREATE/DROP TABLESPACE
        - **SQLite3**: All DDL can be executed inside a transaction except VACUUM
        - **Oracle / SQL Server / MySQL**: DDL must be executed after ending the transaction with COMMIT or ROLLBACK
        - If a transaction is still active when executing DDL, a warning will be displayed
- Table name and column name completion
    - Column name completion works only when the corresponding table name appears to the left of the cursor

![image](./demo.gif)

[Video](https://www.youtube.com/watch?v=_cxBQKpfUds) by [@emisjerry](https://github.com/emisjerry)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | Insert a newline (or execute SQL if line ends with `;` or is a short command[^sc]) |
| `Ctrl`-`Enter`/`J` | Execute SQL |
| `Ctrl`-`F`/`B` | Move Cursor forward or backward |
| `Ctrl`-`N`/`P` | Move Cursor or refer history |
| `Ctrl`-`C` | Exit with rollback |
| `Ctrl`-`D` | Delete character or submit EOF (exit with rollback) |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | Insert the previous SQL (history)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | Insert the next SQL (history) |
| `TAB` | Table name and column name completion |

[^sc]: `DESC`, `EDIT`, `EXIT`, `HISTORY`, `HOST`, `QUIT`, `REM`, `SPOOL`, `START`, `\D`

Supported commands
------------------

- `SELECT` / `INSERT` / `UPDATE` / `DELETE` / `MERGE` ... `;`
    - `INSERT`, `UPDATE`, `DELETE`, and `MERGE` automatically start a transaction.
    - For these commands, input continues across lines until a semicolon (`;`) or the string specified with the `-term` option is entered.
- `COMMIT`
- `ROLLBACK` `;`  -- semicolon required
- `SAVEPOINT savepoint;`  
   (or `SAVE TRANSACTION savepoint;` for Microsoft SQL Server)
- `ROLLBACK TO savepoint;`  
   (or `ROLLBACK TRANSACTION savepoint;` for Microsoft SQL Server)
- `SPOOL`
    - `spool FILENAME` .. open FILENAME and write log and output.
    - `spool off` .. stop spooling and close.
- `EXIT` / `QUIT`
    - Rollback a transaction and exit SQL-Bless.
- `START filename`
    - Start the SQL script given with filename
- `REM comments`
- `DESC [tablename]` / `\D [tablename]`
    - When a table name is specified, shows the schema of that table.
    - If omitted, displays a list of tables.
        - In the table list view, the following keys are available:
        - `r`: Open the `EDIT` command for the selected table.
        - `Enter`: Show the schema of the selected table.
- `HISTORY`
    - Show the history of input SQLs
- `EDIT [tablename [WHERE conditions...]]`
    - Start an [editor][csvi] to modify the selected records of the table.
    - In the editor, these keys are bound.
        - `x` or `d`: set NULL to the current cell
        - `ESC` + `y`: Apply changes and exit
        - `ESC` + `n`: Discard changes and exit
        - `q`: Now equivalent to `ESC`
        - `c`: Apply changes and exit (deprecated)
    - Because the EDIT statement automatically generates SQL from data changed in the editor, it may not be able to properly represent SQL data for special types specific to individual databases. If you find it, we would appreciate it if you could [contact us](https://github.com/hymkor/sqlbless/issues/new).
- `HOST command-line`
    - Executes an operating system command.

&nbsp;

- `;` (or the string specified with `-term string`) is a statement seperator when script is executed
- When sql is input interactively, terminator string (`;` or the string specified with `-term string`) is ignored

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
go install github.com/hymkor/sqlbless/cmd/sqlbless@latest
```

Configuration
-------------

SQL-Bless does **not** rely on any external configuration files. All settings—including database connection parameters, output format, and other options—are specified directly via command-line arguments.

For convenience, you can wrap frequently used connection settings in a shell script or batch file. Sample scripts for various databases are provided in this repository (see `run-*.cmd`). This approach ensures predictable, repeatable startup behavior without relying on hidden configuration files.

How to start
-------------

    $ sqlbless {options} [DRIVERNAME] DATASOURCENAME

DRIVERNAME can be omitted when DATASOURCENAME contains DRIVERNAME.

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- The drivers used are https://github.com/glebarez/go-sqlite
- [Example startup batch file](https://github.com/hymkor/sqlbless/blob/master/run-sqlite3.cmd)

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE
    $ sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- The driver used is https://github.com/sijms/go-ora
- [Example startup batch file](https://github.com/hymkor/sqlbless/blob/master/run-oracle.cmd)

### PostgreSQL

    $ sqlbless postgres host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable
    $ sqlbless postgres postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full
    $ sqlbless postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full

- The driver used is https://github.com/lib/pq
- [Example startup batch file](https://github.com/hymkor/sqlbless/blob/master/run-psql.cmd)

### SQL Server

    $ sqlbless sqlserver sqlserver://@localhost?database=master

( Windows authentication )

    $ sqlbless sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc"

- The driver used is https://github.com/microsoft/go-mssqldb
- [Example startup batch file](https://github.com/hymkor/sqlbless/blob/master/run-mssql.cmd)

### MySQL

    $ sqlbless.exe mysql user:password@/database

- The driver used is http://github.com/go-sql-driver/mysql
- The `?parseTime=true&loc=Local` parameter is preset, but it can be overridden
- [Example startup batch file](https://github.com/hymkor/sqlbless/blob/master/run-mysql.cmd)

Common Options
--------------

- `-crlf`
    - Use CRLF
- `-fs string`
    - Set a field separator (default: `","`)
- `-null string`
    - Set a string representing NULL (default: &#x2400;)
- `-tsv`
    - Use TAB as seperator
- `-f string`
    - Start the script
- `-submit-enter`
    - Submit by [Enter] and insert a new line by [Ctrl]-[Enter]
- `-debug`
    - Print type-information in the header of `SELECT` and `EDIT`
- `-spool filename`
    - Spool to filename from startup
- `-help`
    - Help
* `-rv`
    - Use color schemes designed for light (white) backgrounds.

[csvi]: https://github.com/hymkor/csvi

### Environment Variables

- `NO_COLOR`
  - If defined and non-empty, suppress all color output.
  - If undefined, colors are displayed as usual.
  - (Follows [https://no-color.org/](https://no-color.org/))
- `RUNEWIDTH_EASTASIAN`
  - `"1"` — Treat ambiguous-width Unicode characters as double-width (2 cells).
  - Any other non-empty value — Treat them as single-width (1 cell).
- `COLORFGBG`
  - When defined in the form `(FG);(BG)`, where `(FG)` and `(BG)` are integers representing the foreground and background colors respectively, and `(FG)` is less than `(BG)`, CSVI assumes a light background and uses colors suitable for it (equivalent to `-rv`).

Acknowledgements
-----------------

- [emisjerry (emisjerry)](https://github.com/emisjerry) - [#1],[#2],[Movie]

[#1]: https://github.com/hymkor/sqlbless/issues/1
[#2]: https://github.com/hymkor/sqlbless/issues/2
[Movie]: https://youtu.be/_cxBQKpfUds

Author
------

[hymkor (HAYAMA Kaoru)](https://github.com/hymkor)

License
-------

MIT License
