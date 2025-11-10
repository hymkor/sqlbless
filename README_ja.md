SQL-Bless
=========

<!-- badges.cmd | -->
[![Go Test](https://github.com/hymkor/sqlbless/actions/workflows/go.yml/badge.svg)](https://github.com/hymkor/sqlbless/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-MIT-red)](https://github.com/hymkor/sqlbless/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hymkor/sqlbless.svg)](https://pkg.go.dev/github.com/hymkor/sqlbless)
<!-- -->

[&lt;English&gt;](./README.md) / **&lt;Japanese&gt;**

SQL-Bless は、SQL\*Plus に着想を得た、複数のデータベースエンジンに対応するコマンドライン用 SQL クライアントです。

SQL-Bless は「安全性と再現性」を最優先に設計されています。
かつて、顧客の都合で未完成のシステムを納品せざるを得ず、その後、データ不整合の修正に膨大な労力を費やした経験があります。
その経験から、「人為的ミスや予期せぬ自動コミットを防ぎ、すべての操作を追跡可能にするツール」が必要だと痛感しました。
SQL-Bless は、そうした現場での教訓から生まれた、安全で再現性のある DB メンテナンスツールです。

- 複数行の SQL 入力を編集するための Emacs 風キーバインド
    - Enter キーはデフォルトで改行を挿入します
    - ↑(上)矢印キーまたは Ctrl-P でカーソルを前の行に移動して編集できます
    - Ctrl-J または Ctrl-Enter を押すと、入力を即時に実行します
    - Enter キー単独の場合でも、最終行がセミコロンで終わっている場合や、最初の単語が `EXIT` や `QUIT` といった非SQLコマンドの場合は入力が実行されます
- SELECT結果は CSV 形式で保存
- 以下のRDBMSをサポート
    - SQLite3
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- データベースのレコードをスプレッド風に編集可能 (`EDIT` コマンド)
- トランザクションモード動作（オートコミット無効化）
    - DML（INSERT/UPDATE/DELETE）実行時に自動でトランザクションを開始します
    - ユーザが BEGIN 文を入力することはできません（内部で自動管理されるため）
    - COMMIT または ROLLBACK を実行することでトランザクションを終了できます
    - DDL（CREATE/ALTER/DROP 等）のトランザクション内実行可否はデータベースごとに異なります
        - **PostgreSQL**: VACUUM, REINDEX, CLUSTER, CREATE/DROP DATABASE, CREATE/DROP TABLESPACE を除き、トランザクション内で実行可能
        - **SQLite3**: VACUUM を除き、トランザクション内で実行可能
        - **Oracle / SQL Server / MySQL**: DDL を実行する前に、COMMIT もしくは ROLLBACK でトランザクションを終了させる必要があります
        - DDL 実行時に既存トランザクションが残っている場合は、警告が表示されます。
- テーブル名・カラム名補完
    - ただし、カラム名補完はテーブル名がカーソルより左側に登場している時のみ

![image](./demo.gif)

[@emisjerry](https://github.com/emisjerry) さんによる [紹介動画](https://www.youtube.com/watch?v=_cxBQKpfUds)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | 改行（末尾;または短いコマンド[^sc]時はSQLを実行） |
| `Ctrl`-`Enter`/`J` | SQLを実行 |
| `Ctrl`-`F`/`B` | カーソルを前後に移動 |
| `Ctrl`-`N`/`P` | カーソル移動、もしくはヒストリ参照 |
| `Ctrl`-`C` | ロールバックして終了 |
| `Ctrl`-`D` | 一次削除もしくは、EOF:ロールバックして終了 |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | ヒストリ参照(過去方向)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | ヒストリ参照(未来方向) |
| `TAB` | テーブル名・カラム名補完 |

[^sc]: `DESC`, `EDIT`, `EXIT`, `HISTORY`, `HOST`, `QUIT`, `REM`, `SPOOL`, `START`, `\D`

サポートコマンド
---------------

- `SELECT` / `INSERT` / `UPDATE` / `DELETE` / `MERGE` ... `;`
    - `INSERT`, `UPDATE` , `DELETE` は自動的にトランザクションを開始します
    - これらのコマンドは、セミコロン `;`、もしくは `-term string` で指定された文字列があるまで、Enter を押下しても入力が継続します。
- `COMMIT`
- `ROLLBACK` `;`  -- semicolon required
- `SAVEPOINT savepoint;`  
   (or `SAVE TRANSACTION savepoint;` for Microsoft SQL Server)
- `ROLLBACK TO savepoint;`  
   (or `ROLLBACK TRANSACTION savepoint;` for Microsoft SQL Server)
- `SPOOL`
    - `spool FILENAME` .. FILENAME を開いて、ログや出力を書き込みます
    - `spool off` .. スプールを止めてクローズします
- `EXIT` / `QUIT`
    - トランザクションをロールバックして、SQL-Bless を終了します
- `START filename`
    - ファイル名で指定した SQL スクリプトを実行します。
- `REM comments`
- `DESC [tablename]` / `\D [tablename]`
    - テーブル名が指定された場合、そのテーブルのスキーマを表示します
    - テーブル名が省略された場合、テーブルの一覧を表示します
        - テーブル一覧では以下のキーが拡張されます
        - `r`: そのテーブルのデータの編集モードに入ります
        - `Enter`: そのテーブルのスキーマを表示します
- `HISTORY`
    - 入力履歴を表示します
- `EDIT [tablename [WHERE conditions...]]`
    - 選択したテーブルのレコードを修正するため [エディタ][csvi] を起動します
    - エディタ中では以下のキーが拡張されます
        - `x` or `d`: セルに NULL をセットする
        - `ESC`+`y`: 変更を適用して終了
        - `ESC`+`n`: 修正を破棄して終了
        - `q`: `ESC` と等価
        - `c`: 変更を適用して終了(廃止予定)
    - EDIT文は、エディターでの変更データから自動で SQL を生成する都合、個々のデータベース固有の特殊な型向けの SQL データをうまく表現できない場合があります。見つかりましたら、[ご連絡](https://github.com/hymkor/sqlbless/issues/new)いただけるとたすかります。
- `HOST command-line`
    - OS コマンドを実行します

&nbsp;

- スクリプトを実行する時、セミコロン `;`、もしくは `-term string` で指定された文字列が文の区切りとなります
- インタラクティブに SQL を入力する時、セミコロン`;` もしくは`-term string` で指定された文字列は無視されます

スプールファイルの例
--------------------

``` CSV
# (2023-04-17 22:52:16)
# select *
#   from tab
#  where rownum < 5
TNAME,TABTYPE,CLUSTERID
AQ$_INTERNET_AGENTS,TABLE,<NULL>
AQ$_INTERNET_AGENT_PRIVS,TABLE,<NULL>
AQ$_KEY_SHARD_MAP,TABLE,<NULL>
AQ$_QUEUES,TABLE,<NULL>
# (2023-04-17 22:52:20)
# history
0,2023-04-17 22:52:05,spool hoge
1,2023-04-17 22:52:16,"select *
  from tab
 where rownum < 5"
2,2023-04-17 22:52:20,history
```

インストール
-----------

バイナリパッケージを[Releases](https://github.com/hymkor/sqlbless/releases)よりダウンロードして、実行ファイルを展開してください。

### scoop インストーラーを使用する場合

```
scoop install https://raw.githubusercontent.com/hymkor/sqlbless/master/sqlbless.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install sqlbless
```

### go install でインストールする場合

```
go install github.com/hymkor/sqlbless/cmd/sqlbless@latest
```


設定（Configuration）
--------------------

SQL-Bless は設定ファイルを必要とせず、すべてコマンドラインオプションで指定できます。
デフォルトでは、ローカルのテスト用データベースに接続するための設定は含まれていません。そのため、接続情報は起動時に必ず指定してください。

たとえば、バッチファイルやシェルスクリプトを作成して、SQL-Bless を起動する際に接続情報を渡す方法が推奨されます。

起動方法
--------

    $ sqlbless {options} [DRIVERNAME] DATASOURCENAME

DRIVERNAME は、DATASOURCENAME の中に含まれている場合は省略可能です。

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- 使用ドライバ : https://github.com/glebarez/go-sqlite
- [起動バッチファイル例](https://github.com/hymkor/sqlbless/blob/master/run-sqlite3.cmd)

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE
    $ sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- 使用ドライバー: https://github.com/sijms/go-ora
- [起動バッチファイル例](https://github.com/hymkor/sqlbless/blob/master/run-oracle.cmd)

### PostgreSQL

    $ sqlbless postgres host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable
    $ sqlbless postgres postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full
    $ sqlbless postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full

- 使用ドライバー https://github.com/lib/pq
- [起動バッチファイル例](https://github.com/hymkor/sqlbless/blob/master/run-psql.cmd)

### SQL Server

    $ sqlbless sqlserver sqlserver://@localhost?database=master

( Windows authentication )

    $ sqlbless sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc"

- 使用ドライバー https://github.com/microsoft/go-mssqldb
- [起動バッチファイル例](https://github.com/hymkor/sqlbless/blob/master/run-mssql.cmd)

### MySQL

    $ sqlbless.exe mysql user:password@/database

- 使用ドライバー http://github.com/go-sql-driver/mysql
- パラメータ `?parseTime=true&loc=Local` が予め設定されていますが、上書き可能です
- [起動バッチファイル例](https://github.com/hymkor/sqlbless/blob/master/run-mysql.cmd)

共通オプション
-------------

- `-crlf`
    - 改行コードに CRLF を使う
- `-fs string`
    - 区切り文字を指定する(default: `","`)
- `-null string`
    - NULLを表現する文字列を指定(default: &#x2400;)
- `-tsv`
    - タブを区切り文字に使う
- `-f string`
    - スクリプトを実行する
- `-submit-enter`
    - `Enter` で確定し、`Ctrl`-`Enter` で新しい行を挿入するようにする
- `-debug`
    - `SELECT` と `EDIT` のヘッダに型情報を表示するようにした
- `-spool filename`
    - 指定したファイルに起動時からスプールする
- `-help`
    - ヘルプを表示
- `-rv`
    - 白背景を前提とした色を使用する

環境変数
--------

- `NO_COLOR`
    - 1文字以上設定されていたらカラー表示を抑制する
    - 未定義の場合、通常どおりカラー表示を行う
- `RUNEWIDTH_EASTASIAN`
    - `"1" - Unicodeの曖昧幅の文字は2桁とする
    - それ以外が1文字以上設定れていたら1桁とする
- `COLORFGBG`
    - `(FG);(BG)`形式で、前景色を表す`(FG)`と背景色を表す`(BG)`が整数で `(FG)` が `(BG)` より小さい整数の時、白背景を想定した色使いをするようにする (`-rv`と等価)

Acknowledgements
-----------------

- [emisjerry (emisjerry)](https://github.com/emisjerry) - [#1],[#2],[Movie]

[#1]: https://github.com/hymkor/sqlbless/issues/1
[#2]: https://github.com/hymkor/sqlbless/issues/2
[Movie]: https://youtu.be/_cxBQKpfUds

Author
------

[hymkor (HAYAMA Kaoru)](https://github.com/hymkor)

License
-------

MIT License
