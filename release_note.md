- On start, print version, GOOS, GOARCH, and runtime-version.
- Add the option -null: set a string represeting NULL

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
