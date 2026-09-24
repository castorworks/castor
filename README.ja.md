# Castor

[English](README.md) | [简体中文](README.zh-CN.md) | **日本語** | [한국어](README.ko.md)

**vibe coding のためのフルスタックスキャフォールド。**

Castor は Go + Next.js で構築された、あなたと AI コーディングエージェントのための拡張可能なアプリケーション基盤です。業務要件を自然言語で記述し、既存の認証、権限、ファイル管理、国際化、デプロイの仕組みの上に機能を構築することで、アイデアを動作するフルスタックアプリケーションへと形にできます。

## vibe coding に Castor が適している理由

- **エージェントに明確な指針を与える**：ルートおよびアプリケーションごとの `AGENTS.md` が、アーキテクチャ、コードパターン、API 契約、モジュール追加のチェックリストを定めており、あなたと AI コーディングエージェントが共通して従います。
- **完成した機能から拡張する**：公開サイトと管理画面を備え、ユーザー、ロール、メニュー、アセット、通知、辞書、監査ログを新しい業務モジュールの実装リファレンスとして利用できます。
- **レイヤー横断の制約をチェックに任せる**：単一の検証エントリポイントがフォーマット、静的解析、テスト、フロントエンドビルドをカバーします。ガードレールテストが、アーキテクチャ上の依存関係、RBAC リソースの登録、辞書の契約、4 言語の翻訳をチェックします。
- **開発からデプロイまでを短縮する**：VS Code の F5 でローカル環境を準備して両方のデバッガーを起動でき、Compose / Kubernetes のデプロイとバックアップのツールも同梱しています。

## AI で構築を始める

1. 下記のクイックスタートに従ってプロジェクトを起動し、組み込み機能を試します。
2. AI コーディングエージェントに [AGENTS.md](AGENTS.md) と関連するアプリケーションのガイドを読ませたうえで、業務要件、フィールド、権限、受け入れ条件を伝えます。
3. 既存のモジュールにならってデータベース、API、権限、メニュー、フロントエンド、4 言語の翻訳を実装し、`scripts/check.py` を実行して変更を検証します。

たとえば、次のような依頼から始められます。

```text
AGENTS.md とバックエンド・フロントエンドの開発ガイドを読んでください。
既存のページネーション付き CRUD モジュールにならって、名前・説明・ステータスを
持つプロジェクト管理機能を追加し、一覧、絞り込み、作成、編集、削除に対応
してください。データベースマイグレーション、API エンドポイント、RBAC リソース、
メニュー、ステータス辞書、フロントエンドのページ、中国語・英語・日本語・韓国語の
翻訳、テストも含めてください。scripts/check.py を実行して変更を検証してください。
```

## プロジェクト構成

```text
apps/
├── api/                     # Go API
└── web/                     # Next.js Web アプリケーション

deploy/
├── config/                  # 共用の API 設定テンプレート
├── compose/                 # 単一ホスト、複数インスタンスでの共用、ローカルのインフラ
└── k8s/                     # Kubernetes Kustomize base + overlays

scripts/
├── check.py                 # 開発者と AI エージェント向けの統一検証
├── dev.py                   # ローカル環境の準備（F5 から使用）
├── build-images.py          # イメージのビルド、プッシュ、エクスポート
├── deploy-compose.py        # ローカルおよびリモートへの Compose デプロイ
├── compose-instance.py      # 共用インフラ：インスタンス設定、プロビジョニング、バックアップ/リストア
├── deploy-k8s.py            # Kubernetes の初期化とデプロイ
└── backup.py                # セルフホスト Compose デプロイのバックアップとリストア
```

## 技術スタック

### バックエンド

| コンポーネント | 技術 |
|-----------|------------|
| 言語 | Go 1.26.0+ |
| Web フレームワーク | Gin |
| ORM | GORM |
| データベース | PostgreSQL |
| キャッシュ | Redis |
| オブジェクトストレージ | S3（RustFS 互換） |
| アクセス制御 | RBAC3（ロール階層、SSD、DSD、認可セッション） |
| 認証 | JWT |
| 依存性注入 | Wire |

### フロントエンド

| コンポーネント | 技術 |
|-----------|------------|
| フレームワーク | Next.js 16 (App Router) |
| UI | React 19 + shadcn/ui + Tailwind CSS 4 |
| 状態管理 | Zustand |
| データ取得 | TanStack Query |
| フォーム | TanStack Form |
| テーブル | TanStack Table |

## クイックスタート

### VS Code でデバッグ（F5）

VS Code でリポジトリのルートを開き、推奨の Go 拡張機能をインストールしてから、デバッグ構成 `Castor` を選択して **F5** を押します。事前に Docker Desktop、Go、Node.js 22+、Bun、Python 3.11+、Chrome をインストールしてください。データベースなどのサービスを手動でインストールする必要はありません。

Windows では、Python と一緒に Python ランチャー（`py`）をインストールしてください。VS Code のタスクは、古い Windows Store の `python3` エイリアスを避けるために `py -3` を使用します。選択されるバージョンは `py -3 --version` で確認できます（3.11+ が必要）。以下の手動コマンドでは、`python3` を `py -3` に置き換えてください。

初回起動時には PostgreSQL、Redis、RustFS、ストレージバケット、開発用設定が作成され、依存関係のインストールとデータベースの初期化が行われた後、Go / Next.js のデバッガーが起動してブラウザが開きます。2 回目以降の起動では設定、鍵、データが再利用され、データベースの初期化は繰り返し実行できます。Windows、macOS、Linux で同じ手順で開発できます。Docker Desktop が起動していなければ自動的に起動します。Linux で Docker Engine を使う場合は、サービスを自分で起動してください。

- フロントエンドのデフォルトは `http://127.0.0.1:3000`、API は `http://127.0.0.1:1234` です。ポートが使用中の場合は空いているポートが自動的に選ばれます。実際のアドレスは準備タスクの出力で確認してください。
- デフォルトの管理者は `system` です。初期パスワードは `.local/dev/settings.json` の `admin_password` フィールドに保存されます。既存アカウントのパスワードはリセットされません。
- `.local/dev/` は Git の管理対象外で、開発用の鍵、API 設定、デバッガー用の環境変数を含みます。スクリプトは接続アドレスなどの管理対象フィールドを更新します。その他の API 設定は `api.config.toml` で調整できます。
- **Shift+F5** で両方のデバッガーが停止します。インフラは次回のために稼働したままです。データボリュームを残したままコンテナを停止するには、「タスク: タスクの実行 → Castor: Stop services」または `python3 scripts/dev.py stop` を使用します。
- 環境の準備だけを行う場合は `python3 scripts/dev.py prepare` を実行します。スクリプトは中国語、英語、日本語、韓国語に対応しており、`CASTOR_DEV_LANG` で言語を選択できます。

リポジトリのパスごとに独立した Compose プロジェクトとデータボリュームが使われます。インフラのポートは localhost にのみバインドされ、自動的に割り当てられます。Go のデバッグでは `CASTOR_CONFIG_FILE` で生成された設定ファイルを参照します。

### VS Code を使わない場合

Docker、Go 1.26+、Node.js 22+、Bun、Python 3.11+ が必要です。

```bash
python3 scripts/dev.py prepare     # インフラ、設定、依存関係、init-db
python3 scripts/dev.py run api   # ターミナル 1：Go API
python3 scripts/dev.py run web   # ターミナル 2：Next.js
```

`dev.py run` は `prepare` が選んだポートと接続設定（`.local/dev/debug.env`）で各プロセスを起動します。シェルに依存しないため、Windows、macOS、Linux で同じコマンドが使えます。

## 開発ガイドライン

開発の指針は各階層の AGENTS.md にまとめられており、人間のコントリビューターと AI コーディングエージェント（Claude Code、Codex、Cursor など）が同じ規約を共有します。

- [AGENTS.md](AGENTS.md)：作業原則、エンドツーエンドのモジュール追加チェックリスト、セキュリティ不変条件、API 契約、コミット規約
- [apps/api/AGENTS.md](apps/api/AGENTS.md)：バックエンドのレイヤー、RBAC/メニュー、マイグレーション、テストの規約
- [apps/web/AGENTS.md](apps/web/AGENTS.md)：フロントエンドのデータ層、ページ、権限、i18n、テストの規約
- [deploy/README.ja.md](deploy/README.ja.md)：イメージのビルド、Compose / Kubernetes デプロイ、バックアップとリストア
- [SECURITY.ja.md](SECURITY.ja.md)：セキュリティポリシー

コミット前に統一チェックを実行してください。

```bash
python3 scripts/check.py          # または python3 scripts/check.py api | web | scripts
python3 scripts/check.py vuln     # 依存関係の脆弱性スキャン。リリース前または定期的に実行
```

クローン後、リポジトリのルートで一度 `bun install` を実行すると Git hooks が有効になります。pre-commit はステージされたファイルをフォーマットし、pre-push はフロントエンドに変更がある場合にビルドを行い、変更された Go パッケージをテストします。

## 組み込み機能

- 複数の認証方式（パスワード / メール / 電話番号）
- RBAC3 アクセス制御（複数ロールの継承、SSD/DSD、セッションでのロール有効化）
- ファイル/アセット管理（S3 互換ストレージ）
- 監査ログ
- オンラインセッションと強制サインアウト、パスワードポリシー（長さ・複雑さ・有効期限）
- 部署ツリーとロールのデータ範囲（全データ／所属部署と配下／所属部署／指定部署／本人のみ）
- 動的なシステム設定
- データ辞書
- 通知
- 国際化（中国語 / 英語 / 日本語 / 韓国語）

## ライセンス

[MIT](LICENSE)。フロントエンドは [next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter)（MIT）を改変したものです。サードパーティの表記については [NOTICE](NOTICE) を参照してください。
