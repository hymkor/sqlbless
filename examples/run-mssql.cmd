@rem Sample script to launch SQL-Bless with Microsoft SQL Server
@rem Do NOT include production credentials here.

@setlocal
@set PROMPT=$G$S

sqlbless sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc;"
