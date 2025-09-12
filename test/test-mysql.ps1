Set-PSDebug -Strict
Set-Location (Split-Path $MyInvocation.MyCommand.path)

$testLst = "output.lst"

if ( (Test-Path $testLst) ){
    Remove-Item $testLst
}

$script = `
    "DROP TABLE TESTTBL||" +
    "CREATE TABLE TESTTBL|" +
    "( TESTNO  NUMERIC ,|" +
    "  DTTM    TIMESTAMP(3),|" +
    "  DT      DATE ,|" +
    "  TM      TIME(3) ,|" +
    " PRIMARY  KEY (TESTNO) ); ||" +
    "INSERT INTO TESTTBL VALUES|" +
    "(10,TIMESTAMP '2024-05-25 13:45:33.3+09:00',|" +
    " DATE '2024-05-26',|"+
    " TIME '14:53:26.6') ||" +
    "COMMIT||" +
    "EDIT TESTTBL||" +
    "/10|lr2015-06-07 20:21:22.123+09:00|lr2016-07-08|lr15:54:27.345|cyy" +
    "SPOOL $testLst||" +
    "SELECT * FROM TESTTBL||" +
    "SPOOL OFF ||" +
    "ROLLBACK||" +
    #"DROP TABLE TESTTBL||" +
    "EXIT ||"

$conn = $null
Get-Content .\connections.txt |
Where-Object { $_ -like "*mysql*" } |
ForEach-Object {
    $field = ($_ -split "\|")
    $conn = $field[1]
    Write-Host "Found $conn"
}
if ( -not $conn ){
    Write-Host "Connection String not found"
    exit 1
}

..\sqlbless.exe -auto "$script" mysql "$conn"

$ok = $false

Get-Content $testLst |
Where-Object { $_ -notlike "#*" } |
Select-Object -skip 1 |
ForEach-Object {
    $field = ($_ -split ",")
    if ( $field.Length -lt 4 ){
        Write-Host "NG:" $field.Length
        return
    }
    if ( $field[1] -ne "2015-06-07 20:21:22.123 +09:00" ){
        Write-Host "NG:" $field[1]
        return
    }
    if ( $field[2] -notlike "2016-07-08*" ){
        Write-Host "NG:" $field[2]
        return
    }
    if ( $field[3] -notlike "*15:54:27.345" ){
        Write-Host "NG:" $field[3]
        return
    }
    Write-Host "--> OK"
    exit 0
}
Write-Host "--> NG"
exit 1
