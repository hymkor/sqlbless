v0.12.0
=======
May 29, 2024

- [#1] SQLite3 をサポート。windows-386向けには "mattn/go-sqlite3" を、他の組合せには "glebarez/go-sqlite" を使うようにした。(Thanks to [@emisjerry] and [@spiegel-im-spiegel])
- サポートしていないドライバー名が与えられた時、エラーにならない問題を修正
- ( テストスクリプトが最新仕様に対応していなかった点を修正し、./test へ移動 )

[#1]: https://github.com/hymkor/sqlbless/issues/1
[@emisjerry]: https://github.com/emisjerry
[@spiegel-im-spiegel]: https://github.com/spiegel-im-spiegel

v0.11.0
=======
May 27, 2024

- テーブルデータを CSVI で編集するコマンド: `edit テーブル名 [where ...]` を用意
- START コマンドがエラーメッセージを返さない不具合を修正
- START コマンド用のスクリプトの内容はヒストリに含めないようにした
- `-tsv` オプションを使用すると SELECT の列がすべて連結されてしまう不具合を修正
- (go-multiline-ny) Ctrl-P/N を入力する前のテキストを、ヒストリの最新エントリ扱いにして、失なわれないようにした

v0.10.1
=======
May 9, 2024

- SQL がエラーの時でも CSV ページャが呼ばれる問題を修正
- エスケープシーケンスがスプールファイルに含まれてしまう問題を修正
- `desc TABLE` で TABLE が存在しない時もページャが呼ばれる問題を修正
- Ctrl-D もしくは `exit` で終了した時、EOF がエラーとして表示される問題を修正

v0.10.0
=======
May 8, 2024

- テストやベンチマークのため `-auto` オプションを実装
- [ExpectLua] スクリプトで書かれたテストコードを PowerShell へ置き換えた
- SELECT文の出力のため、[CSVI] をページャーとして使用するようにした

[ExpectLua]: https://github.com/hymkor/expect
[CSVI]: https://github.com/hymkor/csvi

v0.9.0
=======
Sep 4, 2023

- 入力行が `;` で終わっていた場合、Enter キーを入力終結として機能するようにした。

v0.8.0
=======
May 15, 2023

- TABキーで、SELECT や INPUT といったキーワードの入力補完ができるようになった。

v0.7.1
=======
May 4, 2023

- インポートしているライブラリを更新
    - go-readline-ny  from v0.10.1 to v0.11.2
    - go-multiline-ny from v0.6.7  to v0.7.0
        - 行頭にカーソルがあるとき、← や Ctrl-B でも前の行末にカーソルを移動できるようにした。
        - 行末にカーソルがあるとき、→ や Ctrl-F でも次の行頭にカーソルを移動できるようにした。

v0.7.0
=======
Apr 25, 2023

- オプション: `-f -` で標準入力よりスクリプトを読むようにした
- 標準入力が端末ではない時、 go-readline-ny を使わず、標準入力をシーケンシャルに読むようにした
- MySQL をサポート
- （デバッグオプションとして、SELECT結果の各列の型を表示する `-print-type` を追加）

v0.6.0
=======
Apr 22, 2023

- （複数行対応していないので）インクリメンタルサーチの Ctrl-S と Ctrl-R を無効化
- オプション -submit-enter を追加（Enter と Ctrl-Enter を入れ替える）
- PostgrelSQL のコマンド psql もやっていないようなので、エラー時の自動ロールバックを廃止
- ファイルの SQL を実行するコマンド `START filename` とオプション `-f filename` を実装（SQLの終端子は `;` になります）
- コメント用の `REM` 文を追加
- `spool` 文：SQL の末尾に `;` を追加するようにした
- DDL文が成功したときは `Ok` を表示するようにした

v0.5.0
=======
Apr 19, 2023

- `spool` でプログラムのバージョンも記録するようにした
- Microsoft SQL Server もサポート
- 最初の SQL が入力されるまでログインエラーが起きない問題を修正

v0.4.0
=======
Apr 17, 2023

- 起動時にバージョン、ビルド時の GOOS,GOARCH,Goのバージョンを表示するようにした
- オプション追加
    - `-null "string"` : NULL を表現する文字列を設定
    - `-fs "string"` : カンマの代わりの区切り文字を指定
    - `-crlf` : 改行に CRLF を使う
    - `-tsv` : 区切り文字として TAB を使う

v0.3.0
=======
Apr 16, 2023

- select: フィールドが utf8 として妥当な `[]byte` の時、文字列として表示するようにした
- `desc` と `\d`: 引数なしでテーブル一覧表示、引数指定のテーブル仕様表示を実装
- エディター改良: 二重引用符で囲まれたテキストをマゼンタで表示
- spool コマンド改良
    - 引数なしがない場合、スプールを終了するのではなく、現在の状況を表示するようにした
    - コマンドごとにタイムスタンプをスプールファイルに出力するようにした
    - プロンプトにスプール中のファイル名を表示
    - アペンドモードでオープンし、既存のファイルを空にしないようにした

v0.2.0
=======
Apr 15, 2023

- スプールされた SQL の各行の先頭に `#` を挿入 ( `grep -v "^#" FILENAME` で、CSVデータのみが取り出せる）
- [go-readline-ny v0.10](https://github.com/nyaosorg/go-readline-ny/releases/tag/v0.10.1) のための修正（色付けinterfaceの戻り値が int から `readline.ColorSequence` 型にかえる対応）
- Oracle以外ではデフォルトでエラー時に自動でロールバックするようにした（前は PosgreSQL の時はロールバックするようにしていた。要は今後サポート DB が増えたときにどちらがデフォルトにするかという話）
- エラー時、メッセージに `(%T)` （エラーの型）を含むようにした
- `exit` , `quit` , EOF で終了する際、トランザクションを自動的にロールバックするようにした。

v0.1.0
=======
Apr 10, 2023

初版
