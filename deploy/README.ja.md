# デプロイガイド

[English](README.md) | [简体中文](README.zh-CN.md) | **日本語** | [한국어](README.ko.md)

Castor は Docker Compose と Kubernetes の 2 通りのデプロイ方法を提供する。どちらも同じ API / Web イメージを使い、同一ドメインで `/api/v1/*` を API に、その他のリクエストを Web に振り分ける。Dockerfile は `apps/api` と `apps/web` にある。

`scripts/` のスクリプトに必要なのは Python 3.9+（標準ライブラリのみ）と、スクリプトが呼び出すツール（Docker、kubectl、ssh）だけで、Windows、macOS、Linux で同じように動作する。Windows では `python3` の代わりに `py -3` で実行する。`cp` / `chmod` の例は Linux と macOS のホスト向けで、Windows では通常どおりファイルをコピーし、権限は NTFS に任せる。

## リリースモデル

1. 明示的なバージョンタグを付けた 2 つのイメージをビルドする。公開済みのタグは上書きしない。
2. PostgreSQL、Redis、S3 と API 設定を用意する。
3. 現行の API イメージで `init-db` を実行する。
4. 初期化が成功したら API と Web を更新し、準備完了を待つ。

`init-db` は PostgreSQL のトランザクション内で advisory lock を保持し、テーブル構造のマイグレーションを実行したうえで、システム設定、辞書、権限リソース、ロール、メニュー、管理者について追加のみ・変更なしの突き合わせを行う（初回はデプロイ識別子も生成する）。失敗時はロールバックし、非ゼロの終了コードを返す。繰り返し実行しても管理者パスワードはリセットされない。通常の API 起動は既存のデータベースと権限設定を読み取るだけで、マイグレーションは実行しない。

イメージの更新とデータベースの変更は別の操作である。互換性のないデータベース変更を伴う場合は、メンテナンスウィンドウを設けてデータをバックアップすること。ローリングアップデートだけに頼ってはならない。旧イメージが新しいデータベースと互換である保証はない。

## テナント構成の選び方

Castor の 1 デプロイは 1 つの組織に対応する。複数の顧客（テナント）を収容するには、テナントごとに 1 インスタンスを動かす。データベースとロール、バケット、Redis キーのプレフィックス、シークレット、ドメインはテナントごとに独立し、インフラは共用でも専用でもよい（Compose は後述の「複数インスタンスでのインフラ共用」、Kubernetes はテナントごとのインスタンス overlay）。テナント同士がデータベースを共有しないので、どのクエリもテナントをまたいでデータを漏らせない。バックアップ、リストア、移行、削除もテナント単位で行える。

| 要件 | 推奨 |
|------|------|
| 1 つの組織の部署や子会社が自分のデータだけを見る | 1 インスタンス + 部署とロールのデータ範囲 |
| 数社から数十社の顧客で強い分離が必要（官公庁、医療、金融） | テナントごとに 1 インスタンス |
| サインアップで即時開通する数百のセルフサービステナント、テナント横断の課金 | 下流の製品で行レベルのマルチテナントを実装する（提供しない） |

インスタンスごとに一定のリソース（API と Web のプロセス、データベース接続。後述の `max_connections` の注意を参照）がかかり、アップグレードもインスタンスごとに行う。以下の一括コマンドでこれを管理しやすくしている。行レベルのマルチテナント（共有データベースで全行に `tenant_id` を持たせる）は、すべての下流プロジェクトがそのコストとリスクを負うことになるため、意図的に Castor に含めていない。必要な製品は少なくとも次を計画すること：

- 全テーブルへの `tenant_id` 追加、一意制約の複合化（ユーザー名、連絡先、ロール・部署・辞書のコード、設定キー）、どのリポジトリクエリや生 SQL も回避できないテナント条件（GORM のスコープに加え、多層防御として PostgreSQL の行レベルセキュリティ）
- システムテーブルごとに全体共通かテナント別かを決めること（メニュー、リソース、辞書、設定）と、それに合わせたシード照合の作り直し
- テナント単位のアセット重複排除（全体での内容ハッシュは、別テナントが同じファイルを持つことを明かしてしまう）
- サインイン時のテナント識別（ドメインまたはコード）と、JWT、セッション、Redis キー、レート制限へのテナントの反映
- 「全員」宛ての通知、ダッシュボード、データ範囲をテナント内に限定し、テナント横断の運用者は別途認可すること

## イメージのビルド

Docker + Buildx が必要。デフォルトは `linux/amd64` でビルドする。ARM ホストでローカル検証する場合は `PLATFORM=linux/arm64` を設定する。API のビルドはデフォルトで公式の Go/Ubuntu ソースを使う。これらに到達できないビルドマシン（中国国内など）では、スクリプト実行時に環境変数 `GOPROXY`、`UBUNTU_MIRROR`、`UBUNTU_SECURITY_MIRROR` を設定すると、ビルド引数として渡される（例：`GOPROXY=https://goproxy.cn,direct`）。

```bash
# イメージをローカルに読み込む
python3 scripts/build-images.py load castor local

# レジストリにプッシュする（ビルドマシンとデプロイマシンでそれぞれ docker login を実行）
python3 scripts/build-images.py push registry.example.com/castor v1.0.0

# アプリケーションイメージをオフラインで引き渡す
python3 scripts/build-images.py tar castor v1.0.0 deploy/artifacts/castor-v1.0.0.tar.gz
```

Web はデフォルトで Node の Dockerfile を使い、ランタイムとヘルスチェックのコマンドを一致させる。本番のルーティングはエントリポイントが処理するため、環境ごとのドメインをフロントエンドイメージに埋め込む必要はない。`NEXT_PUBLIC_*` をカスタマイズする場合、それは引き続きビルド設定に属する。

## Compose

### 設定

```bash
cp deploy/compose/.env.example deploy/compose/.env
cp deploy/config/api.config.example.toml deploy/compose/config/api.config.toml
chmod 600 deploy/compose/.env
```

`.env` の空の鍵をすべて埋める。JWT は 32 バイト以上を推奨、RSA はちょうど 32 ASCII バイトでなければならず、管理者パスワードは 12 文字以上とする。JWT などの鍵は `python3 -c "import secrets; print(secrets.token_hex(32))"` で生成し、RSA 鍵は同じコマンドの 32 を 16 に変えて生成する。項目ごとに別の値を使うこと。実際の `.env` はコミットしない。

SMS とメールの認証コードは任意である。使わない場合は `.env` の `CASTOR_SMS_*` と TOML の `[Mail].Host` を空のままにしておけば、サービスは通常どおり起動し、ユーザーがその方式を選んだときに未設定である旨が表示される。SMS を有効にするには `CASTOR_SMS_ACCESS_KEY`、`CASTOR_SMS_SECRET_KEY` と TOML の `SignName`、`TemplateCode` をすべて設定する。メールを有効にするには `[Mail]` の Host とアカウントを設定し、パスワードは `CASTOR_MAIL_PASSWORD` で注入する。K8s で必要な場合は同名の変数を `secrets.env` に追加する。

`CASTOR_INSTANCE_ID` はインスタンス識別子（Redis key のプレフィックスと JWT の `aud`）であり、単一インスタンスではデフォルトの `castor` のままでよい。他のデプロイと Redis を共用する場合は後述の「複数インスタンスでのインフラ共用」を参照。`API_IMAGE` / `WEB_IMAGE` はビルド結果と一致させる必要がある。`S3_BUCKET` は API 設定とセルフホストのバケット初期化の両方に使われる。初期化ではプライベートバケットのみを作成する。

API の通常の設定は `config/api.config.toml` に置き、鍵は `.env` から注入する。API は UID 10001 で動作するため、マウントする TOML はこのユーザーが読み取れる必要がある。ファイルには機密でない設定のみを置き、権限を 0644 とすることを推奨する。コンテナのログは stdout と一時ディレクトリに書き出され、長期的なログはプラットフォーム側で収集する。

### 単一ホストでのフルデプロイ

```bash
python3 scripts/deploy-compose.py config full
python3 scripts/deploy-compose.py pull full  # ローカルでビルドしたイメージを使う場合はスキップ
python3 scripts/deploy-compose.py up full
```

PostgreSQL、Redis、RustFS を起動してヘルシーになるのを待ち、バケットを作成し、一度限りのデータベース初期化を実行してから、最後に API、Web、Nginx を起動する。`up full` を繰り返し実行すると初期化も再実行されるため、旧バージョンの終了済みコンテナの成功状態が再利用されることはない。

デフォルトのエントリポイントは `http://localhost:8080`。ポートを公開するのは Nginx のみで、データベースとアプリケーションサービスはホストに直接公開しない。HTTPS はサーバー上の既存の TLS プロキシで終端してこのポートに転送する。デフォルトでは `127.0.0.1` にのみバインドし、TLS プロキシが別のホストにある場合に限り `HTTP_BIND=0.0.0.0` を設定する。開発モード以外ではログイン Cookie に `Secure` が付くため、ブラウザがセッションを保持するのは HTTPS か `http://localhost` のときだけである。`http://<サーバー IP>:8080` のような平文アドレスでログインすると、成功したように見えてログイン画面に戻る。ログイン、アップロード、ダウンロードはすべてエントリポイントのドメインを使う。プロキシは元の Host を保持する必要がある。転送ヘッダーはその信頼境界内で設定し、アップロード上限は 100 MB 以上とし、ストリーミングリクエストに適したタイムアウトとバッファリングを設定する。

### 外部インフラの利用

`.env` のデータベース、Redis、S3 のアドレスと認証情報を編集し、TOML でデータベースの SSL、Redis の TLS/Sentinel/Cluster、S3 の Secure/Region などのオプションを設定する（AWS S3 の `Region` はバケットのリージョンと一致させる必要があり、自動検出はされない）。外部の S3 バケットは事前に作成しておく。

```bash
python3 scripts/deploy-compose.py config external
python3 scripts/deploy-compose.py up external
```

このモードでは外部インフラの作成、停止、バックアップは行わない。Compose の `CASTOR_DB_PORT` でデフォルトの 5432 を上書きできる。複雑な Redis トポロジーでは、設定の検証に使われる Address も併せて設定しておく必要がある。

### 複数インスタンスでのインフラ共用

1 台のホスト上で、同じ PostgreSQL、Redis、RustFS を使って複数の Castor（それぞれ独立した API + Web + ゲートウェイ）を動かす。分離の方法は次のとおり。

| リソース | インスタンスごとに専有するもの | 作成者 |
|------|--------------|----------|
| PostgreSQL | データベース + 同名のロール（`REVOKE CONNECT FROM PUBLIC`。他インスタンスのロールは接続できない） | `compose-instance.py provision` |
| RustFS | バケット + そのバケットのオブジェクトのみ読み書きできるユーザー（他のバケットにはアクセスできず、バケットポリシーも変更できない） | `compose-instance.py provision` |
| Redis | key プレフィックス `CASTOR_INSTANCE_ID`（パスワードは共用、論理的な分離） | API 起動時に確保 |
| 鍵 | `CASTOR_JWT_KEY`、`CASTOR_RSA_SECRET`、`CASTOR_DATA_KEY`、管理者の初期パスワード | `compose-instance.py new` がランダム生成 |

`CASTOR_INSTANCE_ID` は JWT の `aud` でもあり、あるインスタンスが発行した token は別のインスタンスでは無効である。API は起動時に Redis の名前空間を自身のデータベースのデプロイ識別子（`init-db` が生成）に登録する。別のデプロイが誤って同じ `CASTOR_INSTANCE_ID` を使うと、RSA 鍵、辞書キャッシュ、ログインのレート制限を黙って共有するのではなく、理由を示して起動に失敗する。

```bash
# 1. 共用インフラ（一度だけ）
cp deploy/compose/.env.shared.example deploy/compose/.env.shared
chmod 600 deploy/compose/.env.shared      # POSTGRES_PASSWORD、REDIS_PASSWORD、S3_SECRET_KEY を設定
python3 scripts/deploy-compose.py up shared

# 2. インスタンスごと：env（ランダムな鍵、DB 名、バケット）を生成し、必要に応じてイメージ、ポート、CORS ドメインを変更
python3 scripts/compose-instance.py new tenant-a 8081   # → deploy/compose/instances/tenant-a.env
python3 scripts/deploy-compose.py up instance deploy/compose/instances/tenant-a.env
```

`up instance` はデータベースとバケットのプロビジョニング（繰り返し実行可能で、ロールのパスワードと RustFS ユーザーの鍵を env の値に同期する）、`init-db` の実行、アプリケーションの起動を順に行う。管理者の初期パスワードは生成された env ファイルにある。更新とロールバックは単一インスタンスのデプロイと同じで、`full` を `instance <env-file>` に置き換えるだけである。`down instance <env-file>` はそのインスタンスだけを停止する。

すべてのインスタンスを一度に操作する（たとえばリリースを展開する）には一括コマンドを使う。インスタンス識別子の順に処理し、最初の失敗で停止するため、新旧が混在した状態が見過ごされることはない。`--keep-going` は処理を続けるが、終了コードはエラーになる：

```bash
python3 scripts/compose-instance.py status
python3 scripts/compose-instance.py all pull
python3 scripts/compose-instance.py all up
python3 scripts/compose-instance.py all backup
```

- 各インスタンスのゲートウェイはそれぞれ `HTTP_PORT` を公開し、前段の TLS プロキシがドメインごとに振り分ける。**インスタンスごとに独立したドメインまたはサブドメインを使う必要がある**。ログイン状態は Domain 属性のない host-only cookie であるため、同一ドメイン上でパスによってインスタンスを分けると、ログイン状態が互いに上書きされる。
- 共用ネットワーク `SHARED_NETWORK` に参加するのは API のみで、サービス名 `postgres`、`redis`、`rustfs` でインフラにアクセスする。Web とゲートウェイはインスタンス自身のネットワークに留まり、`castor-api` は自インスタンスにのみ解決される。
- すべてのインスタンスが `config/api.config.toml` を共用する（インスタンスごとの差異はすべて env にある）。個別の設定が必要な場合は env で `API_CONFIG_FILE` を設定する。
- PostgreSQL の接続数：API レプリカ 1 つあたり最大 `MaxOpenConns`（デフォルト 25）接続を使うため、インスタンス数 × レプリカ数 × 25 が `max_connections`（デフォルト 100）を超えてはならない。超える場合は `MaxOpenConns` を下げるか上限を引き上げる。
- Redis で（key の分離だけでなく）権限の分離が必要な場合は、外部の Redis を使い、インスタンスごとに ACL ユーザー（`~<instance-id>:* +@all -@dangerous`）を作成して、TOML の `[Redis]` に `Username` を設定する。パスワードは引き続き `CASTOR_REDIS_PASSWORD` で注入する。
- リモートデプロイ（`deploy-compose.py remote`）は単一インスタンスのみに対応する。複数インスタンスの場合は対象ホスト上で上記のコマンドを直接実行する。

### ローカル開発

ローカル開発ではこの節のスクリプトは使わない。`python3 scripts/dev.py prepare` が独立したインフラを起動して設定を生成する。ルートの README を参照。

### リモートデプロイ

まずローカルの `deploy/compose/.env` と `config/api.config.toml` を用意する。スクリプトは Compose の設定と鍵を SSH で転送してリモートホスト上で `docker compose` を実行し、リモートでの初期化が成功した後にアプリケーションを起動する。イメージの自動ビルドは行わない。

```bash
# レジストリ方式
python3 scripts/deploy-compose.py remote registry deploy@example.com /opt/castor full

# tar 方式（アプリケーションイメージは事前にビルド/エクスポートしておく）
python3 scripts/deploy-compose.py remote tar deploy@example.com /opt/castor full deploy/artifacts/castor-v1.0.0.tar.gz
```

対象マシンに必要なのは Docker Compose v2（`up/start --wait` に対応したもの）と tar だけで、Python や Bash は不要。tar パッケージにはアプリケーションイメージのみが含まれる。オフライン環境では Compose マニフェストに記載された PostgreSQL、Redis、RustFS、rc、Nginx のイメージも事前に読み込んでおく必要がある。リモートホストへの初回接続と認証には SSH 自体の仕組みを使う。

### 検証と更新

```bash
curl -f http://localhost:8080/healthz
curl -f http://localhost:8080/api/v1/settings/public
docker compose --project-directory deploy/compose -f deploy/compose/compose.yaml ps
```

API コンテナの readiness チェックは `/ready` にアクセスし、PostgreSQL と Redis を検証する。実際の受け入れ確認では、オブジェクトストレージの経路もカバーするため、ログイン、ファイルのアップロードとダウンロードも行うこと。また自分宛てに通知を送り、ベルがすぐ更新されることを確認する。更新されない場合は前段のプロキシがイベントストリーム（`/api/v1/account/notifications/stream`）をバッファリングしている。

以降のリリースでは 2 つのイメージのバージョンを変更してから `pull`、`up` を実行する。`up` はアプリケーションコンテナを再作成するため、TOML とプロキシ設定の更新が確実に反映される。Compose での更新中は短時間の中断が発生することがある。

アプリケーションのロールバックでは、まずデータベースの互換性を確認し、2 つのイメージ参照と設定を旧バージョンに戻す。

```bash
python3 scripts/deploy-compose.py pull full
python3 scripts/deploy-compose.py rollback full
```

`rollback` はアプリケーションのみを更新し、旧バージョンのデータベース初期化は実行しない。

## Kubernetes

既存のクラスター、kubectl、そのクラスターに対応したイングレスコントローラー、外部の PostgreSQL/Redis/S3、およびイメージレジストリへのアクセス権が必要。このディレクトリではデータベースクラスターとストレージシステムは管理しない。

### 環境の準備

```bash
cp deploy/config/api.config.example.toml deploy/k8s/overlays/staging/api.config.toml
cp deploy/k8s/overlays/staging/secrets.env.example deploy/k8s/overlays/staging/secrets.env
cp deploy/k8s/overlays/staging/bootstrap.env.example deploy/k8s/overlays/staging/bootstrap.env
chmod 600 deploy/k8s/overlays/staging/{secrets,bootstrap}.env
```

本番環境では上記の `staging` を `production` に置き換える。

各 overlay が 1 つのインスタンス（独立した namespace）に相当し、`staging` と `production` はその例である。テナントごとに 1 インスタンスとする場合はインスタンス overlay を生成する。`new` は `production` を `deploy/k8s/instances/<id>/` に複製し、namespace `castor-<id>`、ドメイン、`CASTOR_INSTANCE_ID=<id>` を設定したうえで、JWT 鍵、RSA 鍵、管理者の初期パスワードを生成する。その後 `secrets.env` と TOML に外部の認証情報（独立したデータベースとロール、バケットとアクセスキー）を記入する。空の値があると `apply` は実行を拒否する。`apply-all` / `rollback-all` はすべてのインスタンスを識別子の順に処理し、最初の失敗で停止する（`--keep-going` は処理を続けるが、終了コードはエラーになる）。

```bash
python3 scripts/deploy-k8s.py new tenant-a tenant-a.example.com
python3 scripts/deploy-k8s.py apply tenant-a your-kube-context
python3 scripts/deploy-k8s.py apply-all your-kube-context
```

- TOML：外部サービスのアドレス、データベース名/ユーザー、TLS、S3 のバケット/リージョン、CORS ドメインを設定する。機密フィールドは Secret から注入する。
- `secrets.env`：ランタイムの鍵を設定する。`bootstrap.env` には管理者の初期パスワードのみを含め、初期化 Job だけが使用する。
- `kustomization.yaml`：2 つのイメージのレジストリとバージョン、ドメインを置き換える。クラスターに合わせて Ingress class とプライベートレジストリの `imagePullSecrets` を設定する（アプリケーションと初期化テンプレートの両方に設定が必要）。
- TLS：対象の namespace に `castor-tls` という名前の TLS Secret を事前に用意するか、既存の証明書コントローラーで生成する。
- 実際のイングレスコントローラーに合わせて、100 MB 以上のアップロード上限、ストリーミングリクエスト、タイムアウトを設定する。デフォルトの IngressClass がないクラスターでは class を明示的に指定する必要がある。

ConfigMap / Secret は内容のハッシュを含む名前を使うため、設定が変わると Pod テンプレートが更新され、ローリングアップデートが発生する。実際の設定ファイルと Secret の env ファイルはいずれも Git の管理対象外である。`render` の出力には Secret のデータが含まれるため、コミットしたり公開ログに出したりしないこと。

### デプロイ

```bash
python3 scripts/deploy-k8s.py render staging deploy/artifacts/castor-staging.yaml
# 内容を確認した後、この一時ファイルを削除する
python3 scripts/deploy-k8s.py apply staging your-kube-context
python3 scripts/deploy-k8s.py apply production your-kube-context
```

誤って現在のクラスターを使わないよう、スクリプトは context の明示的な指定を必須とする。namespace の適用、環境のリリースロックの取得、設定と初期化テンプレートの適用、現行バージョンの Job の作成、完了の待機、Deployment/Ingress の適用、ローリングアップデートの待機を順に行う。

`castor-init-db` はスケジュールを無効化した CronJob で、各リリースの Job のテンプレートとしてのみ使われ、定期実行されることはない。初期化がタイムアウトまたは失敗した場合はリリースを中止し、元のアプリケーションの Deployment を維持する。Job のログは調査に使え、Job は 1 日後に自動的に削除される。

同一 namespace でのスクリプトによるリリースは、`castor-release-lock` ConfigMap によって直列化される。プロセスが強制終了されてロックが残った場合は、リリースプロセスと初期化タスクが実行中でないことを確認してから、この ConfigMap を削除する。スクリプトが正常終了した場合はロックが解放される。

staging はレプリカ 1 つ、production は HPA がレプリカ数を管理する（最小 2 つ）。マニフェストには `replicas` を記述しないため、`apply` を繰り返しても HPA がスケールアウトしたレプリカが減らされることはない。API は `/health`、`/ready` とスタートアッププローブを使う。終了猶予期間は 30 秒で、アプリケーションのデフォルトの終了時間 10 秒より長い。リソースの requests/limits は負荷に応じて調整できる。

```bash
kubectl --context your-kube-context -n castor-staging get pods,jobs,ingress
kubectl --context your-kube-context -n castor-staging logs deployment/castor-api
curl -f https://castor-staging.example.com/api/v1/settings/public
```

### 更新とロールバック

overlay のイメージバージョンと設定を更新してから `apply` を実行する。初期化に破壊的な変更が含まれる場合は、事前にメンテナンスウィンドウを設ける。

ロールバックの前に、旧アプリケーションがデータベースを読み取れることを確認し、overlay の旧バージョンのイメージと設定を復元してから、次を実行する。

```bash
python3 scripts/deploy-k8s.py rollback production your-kube-context
```

このコマンドはリリースロックを取得し、指定した設定を適用してアプリケーションを更新する。初期化 Job は作成しない。
`kubectl rollout undo` は Deployment のみを扱い、データベース、Job、Ingress、外部サービスはロールバックしない。過去の ConfigMap/Secret は調査とロールバックのため、現時点では自動削除しない。Deployment の過去のリビジョンから参照されなくなったことを確認してから削除する。

## バックアップとリストア

`scripts/backup.py` は Compose のセルフホストデータのみを扱い、短時間の停止を伴う物理スナップショットとして PostgreSQL、Redis（AOF を含む）、RustFS のすべてのデータボリュームを保存し、加えてデプロイ設定をアーカイブする。外部サービスは変更しない。Compose プロジェクト内の実際のコンテナを参照するため、固定のコンテナ名には依存しない。

```bash
python3 scripts/backup.py create full
python3 scripts/backup.py list
python3 scripts/backup.py restore full snapshot-20260909_120000 --confirm snapshot-20260909_120000
```

バックアップ/リストアの前に Alpine のツールイメージを pull し、その後プロジェクトのサービスを停止する。成功後は元々稼働していたサービスを再開する。バックアップが失敗した場合もサービスは再開されるが、リストアが失敗した場合は、部分的にリストアされたデータを提供しないよう停止したままとなる。

物理リストアには、3 つのインフラコンテナのイメージ ID がバックアップ時と一致している必要がある。データベースのバージョンをまたぐアップグレードには専用の移行ツールを使う。リストア操作は既存のデータボリュームを上書きするため、`--confirm <snapshot>` を明示的に指定した場合にのみ実行される。`config.tar` は照合用であり、既存のデプロイ設定を上書きしない。認証情報を変更した後は、整合性を手動で確認する必要がある。

共用インフラ全体のコールドスナップショットには `full` の代わりに `shared` を使う。これは PostgreSQL、Redis、RustFS を停止するため、その間はすべてのインスタンスが利用できない。事前に各インスタンスを停止しておくこと。単一インスタンスのバックアップやリストアには論理スナップショットを使い、そのインスタンスの API だけを停止する。

```bash
python3 scripts/compose-instance.py backup deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py list deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py restore deploy/compose/instances/tenant-a.env snapshot-20260921_120000 --confirm tenant-a/snapshot-20260921_120000
```

スナップショットには `pg_dump` カスタム形式のデータベース、バケットのミラー、その時点の env ファイルが含まれ、`deploy/backups/instances/<instance-id>/` に置かれる。リストアでは `pg_restore --clean` でそのインスタンスのデータベースを上書きし、`rc mirror --remove` でバケットをスナップショットと一致させる。失敗した場合、API は停止したままとなる。リストアは同じアプリケーションバージョンに対してのみ行う。バージョンをまたぐ場合は、先にリストアしてから `init-db` を実行する。Redis には期限切れになるランタイムデータしかないため、バックアップしない。

スナップショットは `deploy/backups/` に置かれ、データと鍵を含む。現在のユーザーだけが読めるディレクトリとして作成されるので、安全なバックアップ先へ各自で移すこと。K8s と外部データベースでは、各プラットフォームのバックアップ/PITR やオブジェクトストレージのバージョニングを利用し、定期的にリストアを検証する。

## リポジトリの検証

`scripts/check.py`（ルートの AGENTS.md を参照）。このうち `bootstrap` の統合テストは `CASTOR_TEST_POSTGRES_DSN` を使い、これは独立した空のテスト用データベースを指していなければならない。このテストは、2 つの初期化タスクの並行実行、繰り返し実行しても管理者パスワードがリセットされないこと、ロールの割り当てが重複しないことを検証する。
