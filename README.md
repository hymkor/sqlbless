SQL\*Bless
===========

The SQL\*Bless is a command-line database client like SQL*Plus or psql.

- Emacs-like keybindings for inline editing of multiple lines of SQL
- Save the result of SELECT in CSV format

![image](./demo.gif)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **Insert a linefeed** |
| `Ctrl`-`Enter`/`J` | **Execute text as SQL** |
| `Ctrl`-`F`/`B`/`N`/`P` | Editing like Emacs |
| `Ctrl`-`D` | Exit with commit |
| `Ctrl`-`C` | Exit with rollback |
| `ALT`-`P`, `Ctrl`-`Up` | Insert the previous SQL (history)|
| `ALT`-`N`, `Ctrl`-`Down` | Insert the next SQL (history) |

Supported commands:

- SELECT / INSERT / UPDATE / DELETE
    - INSERT, UPDATE and DELETE begin the transaction automatically.
- COMMIT / ROLLBACK
- SPOOL
    - `spool FILENAME` .. open FILENAME and write log and output.
    - `spool off` or `spool` .. stop spooling and close.
- EXIT / QUIT

Semicolon `;` can be omitted.

Supporting DB
-------------

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- On error, your transaction is not rolled back.

### PostgreSQL

    $ sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"

- Autocommit is disabled.  With INSERT, UPDATE, or DELETE, a transaction starts.
- On error, the transaction is rolled back automatically because it aborted.


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
