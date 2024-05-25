$testLst = "output.lst"

if ( (Test-Path $testLst) ){
    Remove-Item $testLst
}

$script = `
    "DROP TABLE TESTTBL||" +
    "CREATE TABLE TESTTBL|" +
    "( TESTNO  NUMERIC ,|" +
    "  DTTM    DATETIME ,|" +
    "  DT      DATE ,|" +
    "  TM      TIME ,|" +
    " PRIMARY  KEY (TESTNO) ); ||" +
    "INSERT INTO TESTTBL VALUES|" +
    "(10,STR_TO_DATE('2024-05-25 13:45:33','%Y-%m-%d %H:%i:%s'),|" +
    " STR_TO_DATE('2024-05-26','%Y-%m-%d'),|"+
    " STR_TO_DATE('14:53:26','%H:%i:%s')) ||" +
    "COMMIT||" +
    "EDIT TESTTBL||" +
    "/10|lr2015-06-07 20:21:22|lr2016-07-08|lr15:54:27|qyy" +
    "SPOOL $testLst||" +
    "SELECT * FROM TESTTBL||" +
    "SPOOL OFF ||" +
    "ROLLBACK||" +
    "DROP TABLE TESTTBL||" +
    "EXIT ||"

$conn = $null
Get-Content .\tstdblst | Where-Object { $_ -like "*mysql*" } | ForEach-Object {
    $field = ($_ -split "\|")
    $conn = $field[1]
    Write-Host "Found $conn"
}
if ( -not $conn ){
    Write-Host "Connection String not found"
    exit 1
}

.\sqlbless.exe -auto "$script" mysql "$conn"

$ok = $false

Get-Content $testLst |
Where-Object { $_ -notlike "#*" } |
Select-Object -skip 1 |
ForEach-Object {
    $field = ($_ -split ",")
    if ( $field.Length -lt 4 ){
        return
    }
    if ( $field[1] -ne "2015-06-07 20:21:22" ){
        return
    }
    if ( $field[2] -notlike "2016-07-08*" ){
        return
    }
    if ( $field[3] -notlike "*15:54:27" ){
        return
    }
    $ok = $true
}

if ( $ok ){
    Write-Host "--> OK"
    exit 0
} else {
    Write-Host "--> NG"
    exit 1
}
