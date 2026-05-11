- github.com/sijms/go-ora/v2 は v2.9.0 に UP してはいけない。
  日時データを取得した時、TZ 情報が UTC になってしまう問題がある。
  解決するまで同バージョンは v2.8.22 に固定すべき
  - https://github.com/hymkor/sqlbless/pull/49
  - https://github.com/sijms/go-ora/issues/687

- Oracle や SQL Server で、TZ が "" (空文字列) になってしまう症状がある。
  これをローカルタイムゾーンで補完すると、MS SQL Server でテストが通らなく
  なってしまう
  - https://github.com/hymkor/sqlbless/pull/49
  - https://github.com/hymkor/sqlbless/pull/52 
