SQL-Bless
===========

**&lt;English&gt;** / [&lt;Japanese&gt;](./README_ja.md)

The SQL-Bless is a command-line database client like SQL\*Plus or psql.

- Emacs-like keybindings for editing multi-line SQL input.
    - Pressing Enter inserts a line break by default.
    - Use the ↑(Up) arrow key or Ctrl-P to move the cursor to the previous line for editing.
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
- Auto commit is disabled.
- Table name and column name completion
    - column name completion works only when the corresponding table name appears to the left of the cursor

![image](./demo.gif)

[Video](https://www.youtube.com/watch?v=_cxBQKpfUds) by [@emisjerry](https://github.com/emisjerry)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **Insert a linefeed** |
| `Ctrl`-`Enter`/`J` or `;`+`Enter`[^semicolon] | **Execute SQL** |
| `Ctrl`-`F`/`B` | Move Cursor forward or backward |
| `Ctrl`-`N`/`P` | Move Cursor or refer history |
| `Ctrl`-`C` | Exit with rollback |
| `Ctrl`-`D` | Delete character or submit EOF (exit with rollback) |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | Insert the previous SQL (history)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | Insert the next SQL (history) |
| `TAB` | Table name and column name completion |

[^semicolon]: `;` or the string specfied with the option `-term string`

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
- `HOST command-line`
    - Executes an operating system command.

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

How to start
-------------

    $ sqlbless {options} [DRIVERNAME] DATASOURCENAME

DRIVERNAME can be omitted when DATASOURCENAME contains DRIVERNAME.

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- The drivers used are https://github.com/glebarez/go-sqlite

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE
    $ sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- The driver used is https://github.com/sijms/go-ora

### PostgreSQL

    $ sqlbless postgres host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable
    $ sqlbless postgres postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full
    $ sqlbless postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full

- The driver used is https://github.com/lib/pq

### SQL Server

    $ sqlbless sqlserver sqlserver://@localhost?database=master

( Windows authentication )

    $ sqlbless sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc"

- The driver used is https://github.com/microsoft/go-mssqldb

### MySQL

    $ sqlbless.exe mysql user:password@/database

- The driver used is http://github.com/go-sql-driver/mysql
- The `?parseTime=true&loc=Local` parameter is preset, but it can be overridden

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

[csvi]: https://github.com/hymkor/csvi

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
