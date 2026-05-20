@echo off
call :"%~1" %2 %3 %4 %5 %6 %7 %8 %9
exit /b

:""
echo run.cmd {sqlite^|sqlserver^|mysql^|oracle^|psql}
exit /b

:"sqlite"
:"sqlite3"
sqlbless %* sqlite3 mytestdb
exit /b

:"mssql"
:"sqlserver"
sqlbless %* "sqlserver://@localhost/SQLExpress?Database=master&protocol=lpc"
exit /b

:"mysql"
sqlbless %* mysql "root:@/mydb"
exit /b

:"oracle"
sqlbless %* oracle://scott:tiger@localhost:1521/xepdb1
exit /b

:"psql"
sqlbless %* postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable
exit /b
