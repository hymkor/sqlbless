@rem Sample script to launch SQL-Bless with SQLite3 local DB
@
@setlocal
@set PROMPT=$G$S
sqlbless %* sqlite3 mytestdb
