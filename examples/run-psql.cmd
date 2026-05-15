@rem Sample script to launch SQL-Bless with a local PostgreSQL DB
@rem Do NOT include production credentials here.

@setlocal
@set PROMPT=$G$S

@rem Examples:
@rem   sqlbless postgres host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable
@rem   sqlbless postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable
@rem Same as psql "host=127.0.0.1 port=5432 user=postgres dbname=postgres sslmode=disable"
sqlbless %* postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable
