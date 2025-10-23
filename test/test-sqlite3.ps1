Set-PSDebug -Strict
Set-Location (Split-Path $MyInvocation.MyCommand.path)

$testLst = "output.lst"

if ( (Test-Path $testLst) ){
    Remove-Item $testLst
}

$script = `
    "CREATE TABLE TESTTBL|" +
    "( TESTNO  NUMERIC ,|" +
    "  DT      CHARACTER VARYING(20) ,|" +
    " PRIMARY  KEY (TESTNO) )||" +
    "INSERT INTO TESTTBL VALUES|",
    "(10,'2024-05-25 13:45:33')||" +
    "COMMIT||" +
    "EDIT TESTTBL||" +
    "/10|lr2015-06-07 20:21:22|cyy" +
    "SPOOL $testLst||" +
    "SELECT * FROM TESTTBL||" +
    "SPOOL OFF||" +
    "ROLLBACK||" +
    "EXIT||"

..\sqlbless.exe -auto "$script" sqlite3 :memory:

$ok = $false

Get-Content $testLst |
Where-Object { $_ -notlike "#*" } |
Select-Object -skip 1 |
ForEach-Object {
    $field = ($_ -split ",")
    if ( $field.Length -ge 2 -and $field[1] -eq "2015-06-07 20:21:22" ){
        Write-Host ("Found {0} --> OK" -f $field[1])
        exit 0
    }
}
Write-Host "--> NG"
exit 1
