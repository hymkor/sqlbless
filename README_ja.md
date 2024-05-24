SQL-Bless
=========

[&lt;English&gt;](./README.md) / **&lt;Japanese&gt;**

SQL-Bless は SQL\*Plus や psql のようなコマンドライン用データベースクライアントです。

- Emacs風キーバインドで複数行のSQLをインラインで編集可能
    - Enter キーは改行を挿入するのみ
    - Ctrl-Enter (Ctrl-J) で入力内容を実行
- SELECT結果は CSV 形式で保存
- 以下のRDBMSをサポート[^anydatabase]
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- トランザクションモード動作(オートコミット無効化)

[^anydatabase]: Go言語の "database/sql" によってサポートされるデータベースであれば、`dbspecs.go` に少量の拡張コードを追加することで利用可能

![image](./demo.gif)

| Key | Binding |
|-----|---------|
| `Enter`, `Ctrl`-`M` | **改行を挿入** |
| `Ctrl`-`Enter`/`J` | **SQLを実行** |
| `Ctrl`-`F`/`B` | カーソルを前後に移動 |
| `Ctrl`-`N`/`P` | カーソル移動、もしくはヒストリ参照 |
| `Ctrl`-`C` | ロールバックして終了 |
| `Ctrl`-`D` | 一次削除もしくは、EOF:ロールバックして終了 |
| `ALT`-`P`, `Ctrl`-`Up`, `PageUp` | ヒストリ参照(過去方向)|
| `ALT`-`N`, `Ctrl`-`Down`, `PageDown` | ヒストリ参照(未来方向) |

サポートコマンド
---------------

- SELECT / INSERT / UPDATE / DELETE
    - INSERT, UPDATE , DELETE は自動的にトランザクションを開始します
- COMMIT / ROLLBACK
- SPOOL
    - `spool FILENAME` .. FILENAME を開いて、ログや出力を書き込みます
    - `spool off` .. スプールを止めてクローズします
- EXIT / QUIT
    - トランザクションをロールバックして、SQL-Bless を終了します
- START filename
    - ファイル名で指定した SQL スクリプトを実行します。
- REM comments

- スクリプトを実行する時、セミコロン `;` が文の区切りとなります
- インタラクティブに SQL を入力する時、セミコロン`;` は無視されます

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

### `go install` を使用する場合

```
go install github.com/hymkor/sqlbless@latest
```

### scoop インストーラーを使用する場合

```
scoop install https://raw.githubusercontent.com/hymkor/sqlbless/master/sqlbless.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install sqlbless
```

起動方法
--------

    $ sqlbless {options} DRIVERNAME "DATASOURCENAME"

### Oracle

    $ sqlbless oracle oracle://USERNAME:PASSWORD@HOSTNAME:PORT/SERVICE

- Use https://github.com/sijms/go-ora

### PostgreSQL

    $ sqlbless postgres "host=127.0.0.1 port=5555 user=USERNAME password=PASSWORD dbname=DBNAME sslmode=disable"

- Use https://github.com/lib/pq

### SQL Server

    $ sqlbless sqlserver "sqlserver://@localhost?database=master"
    ( Windows authentication )

- Use https://github.com/microsoft/go-mssqldb

### MySQL

    $ sqlbless.exe mysql user:password@/database

- Use http://github.com/go-sql-driver/mysql

### 共通オプション

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