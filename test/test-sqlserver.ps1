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
    "  DTTM    DATETIME ,|" +
    "  DT      DATE ,|" +
    "  TM      TIME ,|" +
    "  SDTTM   SMALLDATETIME, |"+
    "  DTTM2   DATETIME2, |" +
    " PRIMARY  KEY (TESTNO) ); ||" +
    "INSERT INTO TESTTBL VALUES|" +
    "(10,CONVERT(DATETIME,'2024-05-25 13:45:33',120),|" +
    " CONVERT(DATE,'2024-05-26',23),|"+
    " CONVERT(TIME,'14:53:26',108),|" +
    " CONVERT(SMALLDATETIME,'2024-05-25 14:53:26',120),|" +
    " CONVERT(DATETIME2,'2024-05-25 14:52:13.133',121)) ||" +
    "COMMIT||" +
    "EDIT TESTTBL||" +
    "/10|lr2015-06-07 20:21:22|" +
    "lr2016-07-08|" +
    "lr15:54:27|" +
    "lr2027-03-04 11:12:13|" +
    "lr2025-05-13 13:51:12.144|" +
    "cyy" +
    "SPOOL $testLst||" +
    "SELECT * FROM TESTTBL||" +
    "SPOOL OFF ||" +
    "ROLLBACK||" +
    "DROP TABLE TESTTBL||" +
    "EXIT ||"

$conn = $null
Get-Content .\tstdblst | Where-Object { $_ -notlike "#*" -and $_ -like "*sqlserver*" } | ForEach-Object {
    $field = ($_ -split "\|")
    $conn = $field[1]
    Write-Host "Found $conn"
}
if ( -not $conn ){
    Write-Host "Connection String not found"
    exit 1
}

..\sqlbless.exe -auto "$script" sqlserver "$conn"

$ok = $false

Get-Content $testLst |
Where-Object { $_ -notlike "#*" } |
Select-Object -skip 1 |
ForEach-Object {
    $field = ($_ -split ",")
    if ( $field.Length -lt 6 ){
        Write-Host $field.Length
        return
    }
    if ( $field[1] -notlike "2015-06-07 20:21:22*" ){
        Write-Host $field[1]
        return
    }
    if ( $field[2] -notlike "2016-07-08*" ){
        Write-Host $field[2]
        return
    }
    if ( $field[3] -notlike "*15:54:27*" ){
        Write-Host $field[3]
        return
    }
    if ( $field[4] -notlike "2027-03-04 11:12:00*" ){
        # SMALLDATETIME does not contain SECOND
        Write-Host $field[4]
        return
    }
    if ( $field[5] -notlike "2025-05-13 13:51:12.144*"){
        Write-Host $field[5]
        return
    }
    Write-Host "--> OK"
    exit 0
}
Write-Host "--> NG"
exit 1
