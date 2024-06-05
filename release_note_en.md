* When the cell validation fails, prompt to modify the input text instead of discarding
* Fix: When `-debug` was specfied, `d` or `x` could clear the debug-header.
* Treat the types including FLOAT, DOUBLE, REAL, SERIAL, YEAR as number

v0.13.0
=======
Jun 4, 2024

* Modify the error message of `desc` with no arguments when no tables exist.  
  `: table not found` → `no tables are found`
* Change the time format of spooled files:  
  `# (2024-05-30 18:15:52)` → `### <2024-05-30 18:46:13> ###`
* Insert blank line before the message `Spooling to '%s'`
* `select` and `edit`: implment `-debug` instead of `-print-type` to insert the type-information into the header
* For types that can store time zones, the time zone is now included in date and time literals
* Support fractional seconds, Oracle TIMESTAMP type, and SQL Server SMALLDATETIME and DATETIMEOFFSET type

### Changes of EDIT command

* Executed SQLs are recorded to spooled file now.
* Print `\n---\n` before SQL is displayed.
* When confirming SQL execution, keys other than `y` and `n` are ignored.
* When SQL fails, ask whether continue(`c`) or abort(`a`)
* Minimal input check is now performed when entering data into cells in the editor.
* `x` and `d` store NULL into the current column
* Fix: `edit` could not be started when no data records were selected.

### Changes from csvi v1.10

* Fix: `o` and `O`: inserted column was always the first one of the new line
* Fix: `O`: the line of cursor is incorrect before new cell text is input
* Header can not be modified now.
* Do not create an empty row at the tail

v0.12.0
=======
May 29, 2024

- [#1] Support SQLite3. For windows-386, use "mattn/go-sqlite3" and for others, "glebarez/go-sqlite" (Thanks to [@emisjerry] and [@spiegel-im-spiegel])
- Fix: error was not displayed even when not supported driver name was given
- (Fixed the problem that the test script was not compatible with the latest specifications and moved it to ./test)

[#1]: https://github.com/hymkor/sqlbless/issues/1
[@emisjerry]: https://github.com/emisjerry
[@spiegel-im-spiegel]: https://github.com/spiegel-im-spiegel

v0.11.0
=======
May 27, 2024

- Create new statement: `edit TABLENAME [WHERE...]` to edit the records of table with [CSVI]
- Fix: The command `START` did not show error-messages
- `start`: do not include the contents of script into history
- `select`: Fix: all columns were joined when `-tsv` was specified
- (go-multiline-ny) The text before the first Ctrl-P/N is treated as if it were the latest entry in the history not to lose them

v0.10.1
=======
May 9, 2024

- Fix: CSV pager was called even when SQL Statement raised error
- Fix: Escape Sequences were inserted into the spooled file
- Fix: `desc TABLE` called pager even when TABLE did not exist
- Fix: EOF was reported as an error when Ctrl-D or `exit` is typed.

v0.10.0
=======
May 8, 2024

- Implement `-auto` option (for test and benchmark)
- Replace the test code written by [ExpectLua]-Script to PowerShell
- Use [CSVI] as a pager for the output of SELECT statement

[ExpectLua]: https://github.com/hymkor/expect
[CSVI]: https://github.com/hymkor/csvi

v0.9.0
======
Sep 4, 2023

- When lines end with `;`, Enter-key works as submiting

v0.8.0
======
May 15, 2023

- Added input completion for some keywords such as SELECT and INSERT. 

v0.7.1
======
May 4, 2023

- Update importing libraries
    - go-readline-ny  from v0.10.1 to v0.11.2
    - go-multiline-ny from v0.6.7  to v0.7.0
        - Ctrl-B can move cursor to the end of the previous line
        - Ctrl-F can move cursor to the beginning of the next line

v0.7.0
======
Apr 25, 2023

- Option `-f -`: read a script from STDIN
- When STDIN is not a terminal, do not use go-readline-ny and read STDIN sequentially
- Support MySQL
- Add debug option -print-type

v0.6.0
======
Apr 22, 2023

- Disable Ctrl-S and Ctrl-R (incremental search)
- Add the option -submit-enter
- Remove automatic-rollback on error because psql (PostgreSQL) does not do it
- Implement `START filename` and `-f filename`
- Implement `REM` for comments
- Spool: append `;` at the tail of SQL
- Print `Ok` after DDL succeeds.

v0.5.0
======
Apr 19, 2023

- `spool` writes program version also
- Support Microsoft SQL Server
- Fix: login error was not raised until the first SQL was input.

v0.4.0
=======
Apr 17, 2023

- On start, print version, GOOS, GOARCH, and runtime-version.
- Add the option -null "string" : set a string represeting NULL
- Add the option -fs "string" : set field separator character instead of comma
- Add the option -crlf: use CRLF for newline
- Add the option -tsv: use TAB as field separator

v0.3.0
======
Apr 16, 2023

- select: when data is []byte and valid as utf8, print it as string
- Implement `desc` and `\d` command to display specifications for the table given as parameter
- Print text enclosed with double quotations with magenta
- Implement `history` command to print command-line histories
- On `spool` command:
    - With no arguments show the current status instead of stopping spooling
    - Output timestamp into the spooling file for each command
    - Show the current spooling filename on prompt
    - Open as append-mode. Do not truncate existing spooled file.

v0.2.0
======
Apr 16, 2023

- Insert `#` at the beginning of each line of spooled SQL
- Fix for go-readline-ny v0.10.1
- Enabled automatic rollback by default on errors except for Oracle
- On error, contain "(%T)" (type of type) into error message
- Implemented automatic rollback of a transaction on 'exit', 'quit', or EOF

v0.1.0
======
Apr 10, 2023

- The first version
