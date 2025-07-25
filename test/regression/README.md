# 回帰テストスイート

このディレクトリには、Sudachi Go実装の回帰テストが含まれています。

## 目的

- 将来の変更による既存機能の破損を早期発見
- 重要な形態素解析結果の一貫性を保証
- Rust実装との互換性維持を確認

## テストファイル

### `morpheme_analysis_test.go`
基本的な形態素解析の回帰テストです。

#### 重要テストケース
- **すもも問題**: `"すもももももももものうち"` の正確な解析
- **基本的な文**: `"東京都に行く"` の形態素分割
- **カタカナ文字**: `"システム"` の処理

### `bocchan_analysis_test.go`
夏目漱石「坊っちゃん」からの文章を使った包括的な回帰テストです。

#### 重要テストケース
- **古典文法**: 明治期の日本語文章での解析精度確認
- **複雑な文**: 長文や複雑な助詞構造の処理
- **固有名詞**: 人名（勘太郎）等の処理
- **感動詞**: 「やーい」等の処理

#### テスト実行方法
```bash
# 全回帰テストの実行
cd test/regression
go test -v

# 基本テストのみ実行
go test -v -run TestMorphemeAnalysisRegression

# 坊っちゃんテストのみ実行
go test -v -run TestBocchanAnalysisRegression

# ベースライン生成（初回または基準更新時）
go test -v -run TestGenerateBaseline
go test -v -run TestGenerateBocchanBaseline
```

#### ベースライン管理
- `baseline.json`: 基本的な解析結果の基準データ
- `bocchan_baseline.json`: 坊っちゃん文章の解析結果基準データ
- 新しい基準を設定する場合は該当の`TestGenerate*Baseline`を実行
- 変更後は必ずベースラインとの差分を確認

## 使用方法

### 新しい機能開発前
```bash
# 現在の状態を確認
go test ./test/regression/...
```

### 開発中
```bash
# 継続的にテスト実行
go test ./test/regression/... -v
```

### 変更完了後
```bash
# 全テストの実行と結果確認
go test ./test/regression/...

# 意図しない変更がある場合はベースライン更新
go test -run TestGenerateBaseline
```

## 注意事項

- テストが失敗した場合は、まず変更内容を確認してください
- 意図的な変更の場合のみベースラインを更新してください
- `"すもももももももものうち"` の解析結果は特に重要です（7形態素期待）

## ディレクトリ構造

```
test/regression/
├── README.md                    # このファイル
├── morpheme_analysis_test.go    # 基本形態素解析テスト
└── baseline.json                # 期待結果のベースライン（自動生成）
```