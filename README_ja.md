SQL-Bless
=========

[&lt;English&gt;](./README.md) / **&lt;Japanese&gt;**

SQL-Bless は SQL\*Plus や psql のようなコマンドライン用データベースクライアントです。

- Emacs風キーバインドで複数行のSQLをインラインで編集可能
    - Enter キーは改行を挿入するのみ
    - Ctrl-Enter (Ctrl-J) で入力内容を実行
- SELECT結果は CSV 形式で保存
- 以下のRDBMSをサポート
    - SQLite3
    - Oracle
    - PostgreSQL
    - Microsoft SQL Server
    - MySQL
- データベースのレコードをスプレッド風に編集可能 (`EDIT` コマンド)
- トランザクションモード動作(オートコミット無効化)

![image](./demo.gif)

[@emisjerry](https://github.com/emisjerry) さんによる [紹介動画](https://www.youtube.com/watch?v=_cxBQKpfUds)

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
- `EDIT tablename [WHERE conditions...]`
    - 選択したテーブルのレコードを修正するため [エディタ][csvi] を起動します
    - エディタ中では `c` キーで変更を適用して終了、`q`キーもしくは`ESC`キーで修正を破棄して終了
    - EDIT文は、エディターでの変更データから自動で SQL を生成する都合、個々のデータベース固有の特殊な型向けの SQL データをうまく表現できない場合があります。見つかりましたら、[ご連絡](https://github.com/hymkor/sqlbless/issues/new)いただけるとたすかります。
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

### SQLite3

    $ sqlbless sqlite3 :memory:
    $ sqlbless sqlite3 path/to/file.db

- Use
    - https://github.com/mattn/go-sqlite3 (Windows-386, TDM-GCC is required)
    - https://github.com/glebarez/go-sqlite (Linux and Windows-amd64)

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
