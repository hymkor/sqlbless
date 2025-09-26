SQL-Bless
=========

[&lt;English&gt;](./README.md) / **&lt;Japanese&gt;**

SQL-Bless は SQL\*Plus や psql のようなコマンドライン用データベースクライアントです。

- 複数行の SQL 入力を編集するための Emacs 風キーバインド
    - Enter キーはデフォルトで改行を挿入します
    - ↑(上)矢印キーまたは Ctrl-P でカーソルを前の行に移動して編集できます
    - Ctrl-J または Ctrl-Enter を押すと、入力を即時に実行します
    - Enter キー単独の場合でも、最終行がセミコロンで終わっている場合や、最初の単語が `EXIT` や `QUIT` といったコマンドの場合は入力が実行されます [^exitquit]
- SELECT結果は CSV 形式で保存
- 以下のRDBMSをサポート
    - SQLite3
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- データベースのレコードをスプレッド風に編集可能 (`EDIT` コマンド)
- トランザクションモード動作(オートコミット無効化)
- テーブル名・カラム名補完
    - ただし、カラム名補完はテーブル名がカーソルより左側に登場している時のみ

[^exitquit]: `EXIT` や `QUIT` で入力集結になるのは v0.21以降

![image](./demo.gif)

[@emisjerry](https://github.com/emisjerry) さんによる [紹介動画](https://www.youtube.com/watch?v=_cxBQKpfUds)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **改行を挿入** |
| `Ctrl`-`Enter`/`J` or `;`+`Enter`[^semicolon] | **SQLを実行** |
| `Ctrl`-`F`/`B` | カーソルを前後に移動 |
| `Ctrl`-`N`/`P` | カーソル移動、もしくはヒストリ参照 |
| `Ctrl`-`C` | ロールバックして終了 |
| `Ctrl`-`D` | 一次削除もしくは、EOF:ロールバックして終了 |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | ヒストリ参照(過去方向)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | ヒストリ参照(未来方向) |
| `TAB` | テーブル名・カラム名補完 |

[^semicolon]: `;` もしくは `-term string` で指定された文字列

サポートコマンド
---------------

- `SELECT` / `INSERT` / `UPDATE` / `DELETE`
    - `INSERT`, `UPDATE` , `DELETE` は自動的にトランザクションを開始します
- `COMMIT` / `ROLLBACK`
- `SPOOL`
    - `spool FILENAME` .. FILENAME を開いて、ログや出力を書き込みます
    - `spool off` .. スプールを止めてクローズします
- `EXIT` / `QUIT`
    - トランザクションをロールバックして、SQL-Bless を終了します
- `START filename`
    - ファイル名で指定した SQL スクリプトを実行します。
- `REM comments`
- `DESC [tablename]` / `\D [tablename]`
    - テーブル名が指定された場合、そのテーブルの使用を表示します
    - テーブル名が省略された場合、テーブルの一覧を表示します
- `HISTORY`
    - 入力履歴を表示します
- `EDIT tablename [WHERE conditions...]`
    - 選択したテーブルのレコードを修正するため [エディタ][csvi] を起動します
    - エディタ中では以下のキーが拡張されます
        = `x` or `d`: セルに NULL をセットする
        - `c`: 変更を適用して終了
        - `q` or `ESC`: 修正を破棄して終了
    - EDIT文は、エディターでの変更データから自動で SQL を生成する都合、個々のデータベース固有の特殊な型向けの SQL データをうまく表現できない場合があります。見つかりましたら、[ご連絡](https://github.com/hymkor/sqlbless/issues/new)いただけるとたすかります。
- `HOST command-line`
    - OS コマンドを実行します

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

起動方法
--------

    $ sqlbless {options} [DRIVERNAME] DATASOURCENAME

DRIVERNAME は、DATASOURCENAME の中に含まれている場合は省略可能です。

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- 使用ドライバ : https://github.com/glebarez/go-sqlite

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE
    $ sqlbless oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- 使用ドライバー: https://github.com/sijms/go-ora

### PostgreSQL

    $ sqlbless postgres host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable
    $ sqlbless postgres postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full
    $ sqlbless postgres://USERNAME:PASSWORD@127.0.0.1:5555/DBNAME?sslmode=verify-full

- 使用ドライバー https://github.com/lib/pq

### SQL Server

    $ sqlbless sqlserver sqlserver://@localhost?database=master

( Windows authentication )

    $ sqlbless sqlserver "Server=localhost\SQLEXPRESS;Database=master;Trusted_Connection=True;protocol=lpc"

- 使用ドライバー https://github.com/microsoft/go-mssqldb

### MySQL

    $ sqlbless.exe mysql user:password@/database

- 使用ドライバー http://github.com/go-sql-driver/mysql
- パラメータ `?parseTime=true&loc=Local` が予め設定されていますが、上書き可能です

共通オプション
-------------

- `-crlf`
    - 改行コードに CRLF を使う
- `-fs string`
    - 区切り文字を指定する(default `","`)
- `-null string`
    - NULLを表現する文字列を指定(default `"<NULL>"`)
- `-tsv`
    - タブを区切り文字に使う
- `-f string`
    - スクリプトを実行する
- `-submit-enter`
    - `Enter` で確定し、`Ctrl`-`Enter` で新しい行を挿入するようにする
- `-debug`
    - `SELECT` と `EDIT` のヘッダに型情報を表示するようにした
- `-help`
    - ヘルプを表示

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
