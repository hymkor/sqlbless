-- This is a test-script for Expect-lua on Windows.
-- ( https://github.com/hymkor/expect )

function run(scripts)
    for i=1,#scripts do
        local s = scripts[i]
        for j=1,#s do
            local rc = expect( j>1 and (j..">") or "SQL>")
            if rc ~= 0 then
                return nil,"can not found prompt"
            end
            send(s[j]..(j<#s and "\r" or "\n"))
        end
    end
    return true
end

function dbTest(arg1,arg2)
    local testLst = "TEST.LST"
    os.remove(testLst)
    local pid,err = assert(spawn("./sqlbless",arg1,arg2))
    if not pid then
        return nil,"can not execute sqlbless"
    end
    local ok,err = run{
        { "CREATE TABLE TESTTBL",
          "(TESTNO NUMERIC ,",
          " TNAME  CHARACTER VARYING(14) ,",
          " LOC    CHARACTER VARYING(13) ) ;" },
        { "INSERT INTO TESTTBL VALUES",
          "(10,'ACCOUNTING','NEW YORK');" },
        { "COMMIT;" },
        { "SPOOL",testLst },
        { "SELECT *","FROM TESTTBL"},
        { "SPOOL","OFF"},
        { "DROP TABLE TESTTBL"},
        { "EXIT" },
    }
    if not ok then
        return nil,err
    end
    wait(pid)

    local lines = {}
    for line in io.lines(testLst) do
        if string.sub(line,1,1) ~= "#" then
            lines[1+#lines] = line
        end
    end
    if #lines < 2 then
        return nil,"too few csv-lines:" .. #lines
    end
    if string.upper(lines[1]) ~= "TESTNO,TNAME,LOC" then
        return nil,"csv: unexpected header:[" .. lines[1].."]"
    end
    if lines[2] ~= "10,ACCOUNTING,NEW YORK" then
        return nil,"csv: unexpected body:[" .. lines[2] .. "]"
    end
    return true
end

function split(s)
    local result = {}
    for p in string.gmatch(s,"[^|]+") do
        result[#result+1] = p
    end
    return result
end

function main(second,dblst)
    timeout = second
    for line in io.lines(dblst) do
        if string.sub(line,1,1) ~= "#" then
            local spec = split(line)
            if #spec < 2 then
                return nil,dblst..": too few arguments: "..line
            end
            print("Try DB:"..spec[1])
            local ok,err = dbTest(spec[1],spec[2])
            if not ok then
                return nil,err
            end
        end
    end
    return true
end

if #arg < 1 then
    print("Usage: expect.lua "..arg[0].." TSTDBLST")
    os.exit(1)
end
local ok,err = main(3.0,arg[1])
if ok then
    print("OK")
    os.exit(0)
else
    print("NG:"..err)
    os.exit(1)
end
