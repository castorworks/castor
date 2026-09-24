# セキュリティポリシー

[English](SECURITY.md) | [简体中文](SECURITY.zh-CN.md) | **日本語** | [한국어](SECURITY.ko.md)

## サポート対象バージョン

セキュリティ修正の対象は、最新リリースと `main` ブランチのみです。

## 脆弱性の報告

セキュリティ上の問題について、公開 issue を**作成しないでください**。

[GitHub Security Advisories](https://github.com/castorworks/castor/security/advisories/new) から非公開で報告してください。影響を受けるバージョンまたはコミット、再現手順、影響範囲を記載してください。

報告は 3 営業日以内の受領確認を目指し、確認された重大度の高い問題には 30 日以内に修正または緩和策を提供することを目指します。報告者が希望しない場合を除き、アドバイザリーに報告者のクレジットを記載します。

## デプロイ時のチェックリスト

- `Development = false` を設定し、すべてのサンプルシークレット（`JwtKey`、RSA 鍵のシークレット、データベース / Redis / S3 の認証情報）を置き換えます。
- `init-db` 用に `CASTOR_DEFAULT_ADMIN_PASSWORD` を設定し、初回ログイン後に管理者パスワードを変更します。
- API と Web の前段で TLS を終端し、ロードバランサーに合わせて `TrustedProxies` を設定します。
- PostgreSQL、Redis、S3 はプライベートネットワーク内に置きます。
