- Option `-f -`: read a script from STDIN
- When STDIN is not a terminal, do not use go-readline-ny and read STDIN sequentially

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
