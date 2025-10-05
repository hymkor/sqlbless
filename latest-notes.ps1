Set-PSDebug -Strict

Get-ChildItem "release_note*" | Sort-Object { Format-Hex -InputObject $_.Name } | ForEach-Object{
    $lang = "(English)"
    if ( $_.FullName -like "*ja*" ) {
        $lang = "(Japanese)"
    }
    $flag = 0
    $section = 0
    Get-Content $_.FullName | ForEach-Object {
        if ( $_ -match "^v[0-9]+\.[0-9]+\.[0-9]+$" ){
            $flag++
            if ( $flag -eq 1 ){
                Write-Host ("`n### Changes in {0} in {1}" -f ($_,$lang))
            }
        } elseif ($flag -eq 1 ){
            if ( $_ -eq "" ){
                $section++
            }
            if ( $section % 2 -eq 1 ){
                Write-Host $_
            }
        }
    }
}

# gist https://gist.github.com/hymkor/50cd1ed60dc94fe50f12658afcb69cbf
