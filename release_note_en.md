- Pressing `r` in the `desc` command without a table name now launches the `edit` command for the table under the cursor.
- Pressing `Enter` in the `desc` command without a table name now launches `desc` command for the table under the cursor.
- Running `edit` without arguments now opens a table selection mode.
- The Oracle `MERGE` statement is now treated as a DML command.
- `COMMIT` and `ROLLBACK` can now be executed without a semicolon (`;`).
- Added support for the [`NO_COLOR`](https://no-color.org/) environment variable to disable colored output.

### Internal changes

- Updated internal dependencies (go-multiline-ny v0.22.1, go-box v3) (#4)
- Removed obsolete go-box v2 references (#4)
- Use context from completion.CmdCompletionOrList.CandidatesContext instead of context.TODO()

v0.24.0
=======
Oct 25, 2025

- Fixed an issue in the `edit` command where table names containing spaces were incorrectly expanded in SQL statements.
- Changed SQLite3 placeholders from `?` to bind variables in the `$v%d` format.
- In interactive mode, SQL-Bless now prevents exiting (`Ctrl-D`, `exit`, `quit`) while a transaction is open, showing: `transaction is not closed. Please Commit or Rollback.` This check is skipped during script execution.
- Modified Ctrl-C behavior during command-line editing. Instead of terminating SQL-Bless, it now cancels (discards) the SQL statement currently being edited.
- Enabled record-style display for SQLite3 `PRAGMA` commands, similar to `SELECT` and `DESC`.
- Enabled syntax highlighting for the `HOST` command and filename completion after `HOST`.

v0.23.0
=======
Oct 14, 2025

- For non-DML SQL, `"database/sql".Conn` is now used instead of `"database/sql".DB`
    - This ensures the same connection is used continuously, avoiding potential issues caused by using different connections for consecutive SQL statements
- Users can no longer use `BEGIN` statements
    - Transactions are automatically started internally, and using `BEGIN` could cause inconsistencies
- Behavior of DDL execution within transactions has been improved
    - Previously, all DDL was disallowed within transactions; now the database-specific rules determine which statements can run
    - **PostgreSQL**: All statements are allowed except `VACUUM`, `REINDEX`, `CLUSTER`, `CREATE/DROP DATABASE`, `CREATE/DROP TABLESPACE`
    - **SQLite3**: All statements are allowed except `VACUUM`
- Update csvi package to v1.15.0 + snapshot:
    - Added key bindings `]` and `[` to adjust the width of the current column (widen and narrow, respectively).
    - Added `-rv` option to prevent unnatural colors on terminals with a white background
    - At startup, the width of ambiguous-width Unicode characters was being measured, but on terminals that do not support the cursor position query sequence `ESC[6n`, this could cause a hang followed by an error. To address this:
        - If `ESC[6n` is not supported, the program now continues without aborting.
        - Skipped the measurement of ambiguous-width Unicode characters when the environment variable `RUNEWIDTH_EASTASIAN` is defined.
    - Suppress color output if the `NO_COLOR` environment variable is set (following https://no-color.org/ )
    - When the environment variable `COLORFGBG` is defined in the form `(FG);(BG)` and `(FG)` is less than `(BG)`, the program now uses color settings designed for light backgrounds (equivalent to `-rv`).

v0.22.0
=======
Oct 5, 2025

- On the `edit` command:
    - Unified the exit operation to the `ESC` key
        - `ESC` + `y`: Apply changes and exit
        - `ESC` + `n`: Discard changes and exit
        - `c`: Still supported but deprecated
        - `q`: Now equivalent to `ESC`
    - Changed the brackets around the table name display from `【】` to `[]`
    - Added options to apply all (`a`) or discard all (`N`) when applying changes
    - Adjusted confirmation SQL output to reduce line usage: SET clause, WHERE clause, and parameter list are now shown on a single line each
- Modified the existing CRLF mode to avoid using golang.org/x/text/transform.

v0.21.0
=======
Sep 27, 2025

- `edit` command
    - Changed to use placeholders for value specification
    - Modified SQLite3 datetime column updates to normalize values in `WHERE` clauses according to column type:
        - `DATETIME` / `TIMESTAMP` columns → `datetime()`
        - `DATE` columns → `date()`
        - `TIME` columns → `time()`
      This ensures updates work regardless of whether ISO8601 strings contain `T` or `Z`
    - Aligned behavior with other commands: if the number of affected rows is zero, no transaction is started and the prompt remains at `SQL>`
    - Removed the behavior where the `edit` command wrote the pre-edit SELECT results to the spool destination.  
      ( The `select` command continues to output to the spool destination as before. )
- In command-line input, pressing Enter alone previously did not terminate input unless the last line ended with a semicolon. This has been changed so that if the input line begins with one of the following command names, it is executed immediately without requiring a semicolon.
    - `DESC`, `EDIT`, `EXIT`, `HISTORY`, `HOST`, `QUIT`, `REM`, `SPOOL`, `START`, `\D`
- Added the `-spool FILENAME` option to enable spooling from startup.
- Added the `host` command to execute operating system commands.

v0.20.0
=======
Sep 14, 2025

- Bug Fixes
    - Fixed an issue where the `EDIT` command failed to update tables containing date columns in SQLite3 databases.
    - Fixed an issue where shared memory connections to SQL Server were not working
        - The required subpackage `"github.com/microsoft/go-mssqldb/sharedmemory"` was not imported
        - When using shared memory connections, the connection string must include the parameter `protocol=lpc`  
          Example: `server=localhost\SQLEXPRESS01;database=master;trusted_connection=yes;protocol=lpc;`  
          Ref: https://github.com/microsoft/go-mssqldb/issues/96
        - Titles were empty when `DESC` or `\D` commands were used without a table name.
- Application Changes
    - Changed the representation of `NULL` from `<NULL>` to the Unicode character U+2400 (&#x2400;, SYMBOL FOR NULL).
    - In the `EDIT` command, setting a non-string cell to an empty string now results in `NULL`.
    - If the first DML affected zero rows, the transaction is not started and the prompt remains `SQL>`.
    - `DESC` and `\D` commands without a table name, and table name completion now exclude non-user tables.
- Library Changes
    - Refactored the codebase: split the main package `"sqlbless"` into subpackages `"dialect"`, `"rowstocsv"`, and `"spread"`.
    - Moved the database-specific customization packages under `"dialect"`.

v0.19.0
=======
Sep 6, 2025

- Parameters for the START command are now completed as filenames.
- EDIT, DESC, \D commands complete a table name now.
- Update the SKK library: go-readline-skk to v0.6.0:
    - Enabled conversion and word registration for words containing slashes in the conversion result
    - Added support for evaluating certain Emacs Lisp forms in conversion results, such as `(concat)`, `(pwd)`, `(substring)`, and `(skk-current-date)` (but not `(lambda)` yet)
- Update the spread library: csvi to v1.14.0:
    - Added search command (`*` and `#`) to find the next occurrence of the current cell's content

v0.18.0
=======
Jun 25, 2025

- Updated go-readline-ny to v1.9.1
- Updated go-multiline-ny to v0.21.0
    - Added yellow syntax highlighting for comments.
    - Switched to using `"go-multiline-ny/completion".CmdCompletionOrList`.
- Added support for table name and column name completion (column name completion works only when the corresponding table name appears to the left of the cursor).
- Fix duplicate reading of script content in START command
- Made it possible to build with Go 1.20.14 to support Windows 7, 8, and Server 2008 or later.

v0.17.0
=======
Jan 20, 2025

- Update the dependency of go-multiline-ny to v0.18.4 and go-readline-ny to v1.7.1
    - When prefix key(Esc) is pressed, echo it as `Esc-`
    - Assign Esc → Enter to submit
- Modified so that a transaction does not start when an error occurs.
- Applied colors to input SQL, such as cyan for reserved words and magenta for strings. 

v0.16.0
=======
Nov 21, 2024

- Show the prompt as `SQL*` instead of `SQL>` during a transaction.
- Erase continuation prompts after submiting so that copied prompt does not get in the
way
- edit: display SQL and usage on the header
- Update go-readline-ny to v1.6.2
    - line-based predictive input support based on history
    - Fix: on Linux desktop, the second or later lines were missing when pasting multi-lines using the terminal feature
- Update go-multiline-ny to v0.17.0
    - Implement the incremental search (Ctrl-R)
    - Fix: on the legacy terminal of Windows, cursor does not move to the upper line
    - Fix: on the terminal of Linux desktop, backspace-key could not remove the line feed
    - Fix: when editing the longer lines than screen height, the number of the lines scrolling was one line short

v0.15.2
=======
Sep 21, 2024

- Fix: [#3] panic occurred during y/n prompts since v0.15.0

[#3]: https://github.com/hymkor/sqlbless/issues/3

v0.15.1
=======
Sep 19, 2024

- Fix: a panic occured when only an empty input was provided
- Separate the main package into cmd/sqlbless to allow usage as a library

v0.15.0
=======
Jul 28, 2024

- With the support for windows/386 in modernc.org/sqlite v1.31.0, the SQLite3 driver has been consolidated to github.com/glebarez/go-sqlite. PureGo implementation is now enabled for all architectures.

v0.14.0
=======
Jun 10, 2024

- When the cell validation fails, prompt to modify the input text instead of discarding
- Treat the types including FLOAT, DOUBLE, REAL, SERIAL, YEAR as number
- Not only the last entry of history, but all modified entries are kept the last value until the current input is completed.
- The the 1st command line parameter DRIVERNAME can be omitted when the 2nd parameter DataSourceName contains DRIVERNAME as the prefix
- To enquote the DATASOURCENAME is now not necessary even when it contains a SPACE
- `desc`: Display the executed sql when `-debug` is specfied
- New option `-term STRING` : specfying the terminater of SQL instead of semicolon  
  ( `-term "/"` enables to execute PL/SQL of Oracle )
- For MySQL, the default setting is now `?parseTime=true&loc=Local`
- `edit`: column names in SQL are now enclosed in double quotes when they contain spaces

### Fixed bugs

- Fix: `edit` with `-debug` would panic when ColumnType.ScanType() returned nil
- Fix: When `-debug` was specfied, `d` or `x` could clear the debug-header.

v0.13.0
=======
Jun 4, 2024

- Modify the error message of `desc` with no arguments when no tables exist.  
  `: table not found` → `no tables are found`
- Change the time format of spooled files:  
  `# (2024-05-30 18:15:52)` → `### <2024-05-30 18:46:13> ###`
- Insert blank line before the message `Spooling to '%s'`
- `select` and `edit`: implment `-debug` instead of `-print-type` to insert the type-information into the header
- For types that can store time zones, the time zone is now included in date and time literals
- Support fractional seconds, Oracle TIMESTAMP type, and SQL Server SMALLDATETIME and DATETIMEOFFSET type

### Changes of EDIT command

- Executed SQLs are recorded to spooled file now.
- Print `\n---\n` before SQL is displayed.
- When confirming SQL execution, keys other than `y` and `n` are ignored.
- When SQL fails, ask whether continue(`c`) or abort(`a`)
- Minimal input check is now performed when entering data into cells in the editor.
- `x` and `d` store NULL into the current column
- Fix: `edit` could not be started when no data records were selected.

### Changes from csvi v1.10

- Fix: `o` and `O`: inserted column was always the first one of the new line
- Fix: `O`: the line of cursor is incorrect before new cell text is input
- Header can not be modified now.
- Do not create an empty row at the tail

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
