# Rust版 vs Go版 Roman Numeral処理の完全比較

## 概要

RustとGoの両版におけるRoman numeral処理の詳細な比較分析。

## 処理メカニズムの比較

### Rust版の処理 (default_input_text plugin)

#### 1. 設定による制御
```
rewrite.def:
Ⅰ Ⅱ Ⅲ Ⅳ Ⅴ ...  (ignore normalize list)
ⅰ ⅱ ⅲ ⅳ ⅴ ...  (ignore normalize list)
```

#### 2. 処理ロジック
```rust
let need_lowercase = ch.is_uppercase();
let need_nkfc = !self.should_ignore(ch) && is_nfkc_quick(...) != IsNormalized::Yes;

match (need_lowercase, need_nkfc) {
    (true, false) => ch.to_lowercase(),  // Roman numeralの場合
    (false, false) => continue,          // 何もしない
    // ...
}
```

#### 3. 実際の動作
- `should_ignore('Ⅲ')` = **true** (rewrite.defで指定)
- `need_nkfc` = **false** (ignoreによりスキップ)
- **結果**: `Ⅲ` → `ⅲ` (大文字→小文字のみ)

### Go版の処理 (DefaultInputTextPlugin)

#### 1. ハードコード制御
```go
func (p *DefaultInputTextPlugin) isUpperRomanNumeral(ch rune) bool {
    return ch >= 'Ⅰ' && ch <= 'Ⅿ'  // U+2160-U+216F
}
```

#### 2. 処理ロジック
```go
// 3. Roman numeral専用処理
if !needsChange && p.isUpperRomanNumeral(ch) {
    lowered := unicode.ToLower(ch)
    replacement = string(lowered)
    needsChange = true  // 重要：後続処理をスキップ
}

// 5. NFKC正規化 (needsChange=trueによりスキップされる)
if !needsChange && needNFKC && !p.ignoreNormalizeSet[ch] {
    // この処理は実行されない
}
```

#### 3. 実際の動作
- `isUpperRomanNumeral('Ⅲ')` = **true**
- `needsChange` = **true** (Roman numeral処理により設定)
- **結果**: `Ⅲ` → `ⅲ` (大文字→小文字のみ)

## 重要な共通点

### 1. 最終結果は同一
**両版とも**: `Ⅲ` → `ⅲ` (Roman numeral形式を保持)

### 2. NFKC正規化の回避
- **Rust版**: `should_ignore()`による明示的回避
- **Go版**: 優先順位制御による暗黙的回避

### 3. 設計思想
両版とも以下の理念を共有：
- Roman numeralの視覚的特性を保持
- 大文字・小文字の統一
- NFKC正規化（`ⅲ` → `"iii"`）の回避

## 実装方式の違い

| 項目 | Rust版 | Go版 |
|------|--------|------|
| 制御方法 | 外部設定ファイル (`rewrite.def`) | ハードコード判定 |
| 回避方法 | `ignore_normalize_set` | 処理優先順位 |
| 拡張性 | 高い（設定変更で対応） | 低い（コード変更が必要） |
| 保守性 | 高い（設定とロジック分離） | 中程度（専用ロジック） |

## 技術的詳細

### Unicode変換の共通性
```
Ⅲ (U+2162) → ⅲ (U+2172)
```
- 両版とも標準Unicode ToLower変換を使用
- マッピングは完全に同一

### NFKC正規化について
```
理論的変換: ⅲ → "iii" (ラテン文字)
実際の動作: ⅲ → ⅲ (Roman numeral保持)
```
- 両版ともNFKC正規化を意図的に回避
- Roman numeralの形を保持する設計

## 設計判断の評価

### 正当性
1. **視覚的一貫性**: Roman numeralの特殊な視覚的意味を保持
2. **検索精度**: 専用辞書エントリとの整合性確保
3. **文化的配慮**: 日本語文書でのRoman numeral使用慣習に配慮

### 実装品質
- **Rust版**: 設定ベースの柔軟なアプローチ
- **Go版**: 専用ロジックによる確実なアプローチ
- **両版**: 結果の一貫性を確保

## 結論

**両版は異なる実装アプローチを取りながら、同一の処理結果を実現している。** 

- **Rust版**: 設定ベースの汎用的アプローチ
- **Go版**: 専用ロジックによる特化アプローチ

いずれの実装も技術的に適切であり、Roman numeralという特殊な文字種の性質を適切に処理している。

## ユーザー理解への修正

**元の理解**: 「大文字→小文字→NFKC正規化」
**実際の動作**: 「大文字→小文字（NFKC正規化は意図的に回避）」

この差異は、Roman numeralの特殊性と日本語形態素解析における設計思想の結果である。