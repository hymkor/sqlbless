SQL\*Bless
===========

The Command-line Database Client

![image](./demo.gif)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | Insert a linefeed |
| `Ctrl`-`Enter`/`J` | Execute text as SQL |
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

Oracle
-------

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

PostgreSQL
----------

    $ sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"
