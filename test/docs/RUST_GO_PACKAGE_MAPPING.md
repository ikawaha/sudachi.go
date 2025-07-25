# Rust版とGo版のパッケージ構造対応表

## パッケージ構造比較

### Rust版の構造 (sudachi.rs/sudachi/src/)

```
src/
├── analysis/                    # 解析エンジン
│   ├── mod.rs
│   ├── stateless_tokenizer.rs
│   ├── stateful_tokenizer.rs
│   ├── lattice.rs              # → Go: lattice/
│   ├── node.rs                 # → Go: lattice/
│   ├── morpheme.rs             # → Go: lattice/
│   ├── mlist.rs                # → Go: lattice/
│   ├── created.rs              # → Go: types/
│   └── inner.rs
├── dic/                        # 辞書関連
│   ├── mod.rs                  # → Go: dic/
│   ├── dictionary.rs           # → Go: dic/
│   ├── grammar.rs              # → Go: dic/
│   ├── lexicon/                # → Go: dic/
│   ├── lexicon_set.rs          # → Go: dic/
│   ├── storage.rs              # → Go: dic/
│   ├── word_id.rs              # → Go: dic/
│   └── ...
├── input_text/                 # 入力処理
│   ├── mod.rs                  # → Go: input/
│   └── buffer/                 # → Go: input/
├── plugin/                     # プラグインシステム
│   ├── mod.rs                  # → Go: plugin/
│   ├── input_text/             # → Go: plugin/input_text/
│   ├── oov/                    # → Go: plugin/oov/
│   ├── path_rewrite/           # → Go: plugin/path_rewrite/
│   └── connect_cost/           # → Go: plugin/connect_cost/
├── config.rs                   # → Go: config/
├── error.rs                    # → Go: 各パッケージのエラー処理
├── util/                       # ユーティリティ
└── lib.rs                      # エントリーポイント
```

### Go版の構造

```
/
├── analysis/                   # トークナイザーのみ
│   ├── tokenizer.go           # Rust: stateless_tokenizer.rs + stateful_tokenizer.rs
│   ├── builder.go             # Go独自のビルダーパターン
│   └── mode.go                # Rust: 分散していたMode定義を統合
├── lattice/                   # Rust: analysis/内から分離
│   ├── lattice.go             # Rust: analysis/lattice.rs
│   ├── node.go                # Rust: analysis/node.rs  
│   ├── node_result.go         # Rust: analysis/morpheme.rs相当
│   ├── split.go               # Go独自の分割処理
│   └── pool.go                # Go独自のメモリプール
├── input/                     # Rust: input_text/
│   ├── buffer.go              # Rust: input_text/buffer/mod.rs
│   ├── normalizer.go          # Go独自の正規化処理
│   └── ...
├── dic/                       # Rust: dic/と同様
│   └── ...
├── plugin/                    # Rust: plugin/と同様
│   └── ...
├── config/                    # Rust: config.rs
│   └── config.go
└── types/                     # Go独自、共通型定義
    └── created_words.go       # Rust: analysis/created.rs
```

## 主要な対応関係

### 1. analysis パッケージ

| Rust版 | Go版 | 説明 |
|--------|------|------|
| `analysis/mod.rs` | `analysis/tokenizer.go` | 公開API |
| `analysis/stateless_tokenizer.rs` | `analysis/tokenizer.go` | ステートレストークナイザー |
| `analysis/stateful_tokenizer.rs` | `analysis/tokenizer.go` | ステートフルトークナイザー |
| `analysis/lattice.rs` | `lattice/lattice.go` | **分離** |
| `analysis/node.rs` | `lattice/node.go` | **分離** |
| `analysis/morpheme.rs` | `lattice/node_result.go` | **分離** |
| `analysis/mlist.rs` | `lattice/` (MorphemeList) | **分離** |
| `analysis/created.rs` | `types/created_words.go` | **分離** |

### 2. 分離された理由

**Go言語の制約による分離:**

1. **Visibilityの制約**
   - Rustは`pub(crate)`で同一クレート内公開が可能
   - Goは大文字開始でパッケージ外公開、小文字でパッケージ内限定
   - 細かい制御が困難なため、パッケージレベルで分離

2. **循環依存の回避**
   - Rustはモジュール内での相互参照が可能
   - Goはパッケージレベルでの厳格な循環依存チェック
   - `lattice`、`input`、`plugin`を独立パッケージとして分離

3. **型エイリアスによる互換性**
   ```go
   // analysis/tokenizer.go - 後方互換性のための型エイリアス
   type Lattice = lattice.Lattice
   type Node = lattice.Node
   type NodeResult = lattice.NodeResult
   type MorphemeList = lattice.MorphemeList
   ```

## analysisパッケージの現在の機能範囲

### ✅ 現在analysisパッケージに残っている機能

1. **コアTokenizer実装**
   - `Tokenizer`構造体とその基本メソッド
   - トークン化処理のメインロジック (`Tokenize`メソッド)
   - モード管理 (`SetMode`, `Mode`)
   - デバッグ機能 (`SetDebugMode`)

2. **プラグイン管理**
   - `PluginManager`構造体
   - プラグインの実行制御
   - プラグイン間の連携処理

3. **解析フロー制御**
   - 入力前処理 → ラティス構築 → 最適パス探索 → 後処理の統合
   - エラーハンドリングとフォールバック処理
   - 各コンポーネントの協調動作

4. **ファクトリー機能**
   - `TokenizerBuilder`によるトークナイザー構築
   - `BuildFromResourceDir`による自動設定
   - プラグインの自動読み込みと設定

### 🚫 外部パッケージに分離された機能

1. **lattice/** - ラティス・ノード・形態素管理
   - `Lattice`: Viterbi解析用ラティス構造
   - `Node`: ラティスノード
   - `NodeResult`: 解析結果ノード（形態素情報）
   - `MorphemeList`: 形態素リスト
   - メモリプール管理

2. **input/** - 入力テキスト処理
   - `InputBuffer`: 入力テキストバッファ
   - `Normalizer`: テキスト正規化
   - 文字カテゴリ処理

3. **plugin/** - プラグインシステム
   - 各種プラグイン実装
   - プラグインインターフェース定義
   - プラグイン固有の処理ロジック

4. **dic/** - 辞書システム
   - 辞書読み込み・管理
   - 文法・語彙情報
   - 単語検索機能

5. **types/** - 共通型定義
   - `CreatedWords`: 作成された単語の追跡
   - その他共通データ型

## Go版の設計方針

### 1. analysisパッケージの役割

**「トークナイザーのオーケストレーター」**
- 各コンポーネントを統合して解析処理を実行
- プラグインシステムの制御
- エラーハンドリングとフォールバック
- 設定管理とファクトリー機能

### 2. パッケージ間の依存関係

```
analysis (オーケストレーター)
├── lattice (データ構造)
├── input (前処理)  
├── plugin (拡張機能)
├── dic (辞書)
├── config (設定)
└── types (共通型)
```

**依存の方向**: `analysis` → 各専門パッケージ（一方向）

### 3. 型エイリアスによる後方互換性

```go
// 利用者は analysis パッケージのみをimportすれば使用可能
import "github.com/ikawaha/sudachi.go/analysis"

// 内部では各専門パッケージの型をエイリアスとして提供
tokenizer := analysis.NewTokenizer(dict)
morphemes := tokenizer.Tokenize("東京に行く") // *analysis.MorphemeList
```

## リファクタリング方針

### analysisパッケージでリファクタリング対象となる機能

1. **tokenizer.goの分割** (1422行 → 複数ファイル)
   - コア機能の整理
   - 関連機能のグループ化
   - テスタビリティの向上

2. **統一API設計**
   - ファクトリー関数の統一
   - オプションパターンの導入
   - エラーハンドリングの一貫性

3. **テスト強化**
   - `tokenizer_test.go`の新規作成
   - Rust版との互換性テスト
   - 各機能の単体テスト

### 外部パッケージとの関係は維持

- 現在の良好な依存関係を保持
- 循環依存を引き起こす変更は避ける
- 既存の型エイリアスによる互換性を維持

---

**この分析により、Go版のanalysisパッケージはトークナイザーのオーケストレーション機能に特化しており、Goの言語制約に適応した適切な設計となっていることが確認できました。**