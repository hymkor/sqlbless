@rem Sample script to launch SQL-Bless with MySQL
@rem Do NOT include production credentials here.

@setlocal
@set PROMPT=$G$S

@rem Example connection: root:@/mydb
sqlbless %* mysql "root:@/mydb"

