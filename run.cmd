@setlocal
@set PROMPT=$G$S
@call :"%~1" %2 %3 %4 %5 %6 %7 %8 %9
@endlocal
@exit /b

:"sqlite"
:"sqlite3"
sqlbless %* sqlite3 mytestdb
@exit /b

:"mssql"
:"sqlserver"
sqlbless %* sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc;"
@exit /b

:"mysql"
sqlbless %* mysql "root:@/mydb"
@exit /b

:"oracle"
sqlbless %* oracle://scott:tiger@localhost:1521/xepdb1
@exit /b

:"psql"
@rem Driver: https://github.com/lib/pq
@rem Schema postgresql://${username}:${password}@localhost:${port}/${database}?options=--search_path%3D${schema}"
sqlbless %* postgres://postgres@127.0.0.1:5432/chinook?sslmode=disable
@exit /b

:"-psql"
psql -h 127.0.0.1 -p 5432 -d postgres -U postgres
@exit /b

