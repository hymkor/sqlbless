@rem Sample script to launch SQL-Bless with Oracle DB

@setlocal
@set PROMPT=$G$S
@rem $ sqlbless oracle://USERNAME:PASSWORD@HOST:PORT/SERVICE
sqlbless %* oracle://scott:tiger@localhost:1521/xepdb1
