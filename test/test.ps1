Set-PSDebug -Strict
Set-Location (Split-Path $MyInvocation.MyCommand.path)

function DB-Test($arg1,$arg2){
    $testLst = "TEST.LST"
    if ( (Test-Path $testLst) ){
        Remove-Item $testLst
    }
    $script = @(
        @( "CREATE TABLE TESTTBL",
           "(TESTNO NUMERIC ,",
           " TNAME  CHARACTER VARYING(14) ,",
           " LOC    CHARACTER VARYING(13) ) ;" ),
        @( "INSERT INTO TESTTBL VALUES",
           "(10,'ACCOUNTING','NEW YORK');" ),
        @( "COMMIT;" ),
        @( "SPOOL",$testLst ),
        @( "SELECT *","FROM TESTTBL" ),
        @( "SPOOL","OFF" ),
        @( "DROP TABLE TESTTBL" ),
        @( "EXIT" )
    )

    $script = ( $script | ForEach-Object{ $_ -join "|"} ) -join "||"
    # Write-Host $script
    ..\sqlbless -auto "$script" "$arg1" "$arg2"

    $lines = ( Get-Content $testLst | Where-Object{ $_ -notlike "#*" } )
    # Write-Host ($lines -join "`n")
    if ( $lines.Length -lt 2 ) {
        Write-Error ("too few csv-lines: {0}" -f $lines.Length)
        exit 1
    }
    if ( $lines[0] -ne "TESTNO,TNAME,LOC" ){
        Write-Error ("csv: unexpected header: {0}" -f $lines[0])
        exit 1
    }
    if ( $lines[1] -ne "10,ACCOUNTING,NEW YORK" ){
        Write-Error ("csv: unexpected body: {0}" -f $lines[1])
        exit 1
    }
}

function main($dblst){
    Get-Content "tstdblst" | ForEach-Object {
        if ( $_ -notlike "#*" ){
            # Write-Host ("LINE: {0}" -f $_)
            $spec = ($_ -split "\|")
            if ( $spec.Length -lt 2 ){
                Write-Error ("too few argument: {0}: {1}" -f ($dblst,$_))
                exit 1
            }
            DB-Test $spec[0] $spec[1]
        }
    }
}
main $args[0]
Write-Host "--> OK"
exit 0
