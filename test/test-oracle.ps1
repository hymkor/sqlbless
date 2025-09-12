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
    "  DT      DATE ,|" +
    "  ST      TIMESTAMP,|" +
    " PRIMARY  KEY (TESTNO) ); ||" +
    "INSERT INTO TESTTBL VALUES|",
    "(10,TO_DATE('2024-05-25 13:45:33','YYYY-MM-DD HH24:MI:SS'),|" +
    " TO_TIMESTAMP('2024-07-08 17:18:19.8787','YYYY-MM-DD HH24:MI:SS.FF')) ||" +
    "COMMIT||" +
    "EDIT TESTTBL||" +
    "/10|lr2015-06-07 20:21:22|lr2024-08-09 10:11:12.7878 +09:00|cyy" +
    "SPOOL $testLst||" +
    "SELECT * FROM TESTTBL||" +
    "SPOOL OFF ||" +
    "ROLLBACK||" +
    "DROP TABLE TESTTBL||" +
    "EXIT ||"

$conn = $null
Get-Content .\connections.txt |
Where-Object { $_ -notlike "#*" -and $_ -like "*oracle*" } |
ForEach-Object {
    $field = ($_ -split "\|")
    $conn = $field[1]
    Write-Host "Found $conn"
}
if ( -not $conn ){
    Write-Host "Connection String not found"
    exit 1
}

..\sqlbless.exe -auto "$script" oracle "$conn"

$ok = $false

Get-Content $testLst |
Where-Object { $_ -notlike "#*" } |
Select-Object -skip 1 |
ForEach-Object {
    $field = ($_ -split ",")
    if ( $field.Length -lt 3 ){
        Write-Host $field.Length
        return
    }
    if ( $field[1] -ne "2015-06-07 20:21:22 +09:00" ){
        Write-Host $field[1]
        return
    }
    if ( $field[2] -ne "2024-08-09 10:11:12.7878 +09:00" ){
        Write-Host $field[2]
        return
    }
    Write-Host ("Found {0} and {1} --> OK" -f ($field[1],$field[2]))
    exit 0
}
Write-Host "--> NG"
exit 1
