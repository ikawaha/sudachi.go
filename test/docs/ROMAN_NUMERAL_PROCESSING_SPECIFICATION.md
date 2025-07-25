# Roman Numeral処理の完全技術仕様書

## 概要

Go版SudachiのDefaultInputTextPluginにおけるRoman numeral処理の詳細な技術仕様を記載する。

## 重要な発見：ユーザー理解との差異

**ユーザーの理解**: 「大文字から小文字に変換してからNFKC正規化」

**実際の実装**: 「大文字から小文字に変換し、NFKC正規化はスキップ」

## 処理フロー

```
入力文字: Ⅲ (U+2162, 大文字Roman numeral)
    ↓
[1] isUpperRomanNumeral() 判定
    ↓ (true)
[2] unicode.ToLower() 実行
    ↓
出力文字: ⅃iii (U+2172, 小文字Roman numeral)
    ↓
[3] NFKC正規化判定
    ↓ (needsChange=true のためスキップ)
[4] 辞書引き実行
    ↓
辞書検索キー: "ⅲ" (Roman numeral形式)
```

## 実装詳細

### 1. Roman Numeral判定

```go
func (p *DefaultInputTextPlugin) isUpperRomanNumeral(ch rune) bool {
    // Unicode range for uppercase roman numerals (U+2160-U+216F)
    return ch >= 'Ⅰ' && ch <= 'Ⅿ'
}
```

**判定範囲**:
- U+2160 (Ⅰ) から U+216F (Ⅿ) まで
- 16個の大文字Roman numeralが対象

### 2. Unicode変換マッピング

| 大文字 | Unicode | 小文字 | Unicode | 
|--------|---------|--------|---------|
| Ⅰ      | U+2160  | ⅰ      | U+2170  |
| Ⅱ      | U+2161  | ⅱ      | U+2171  |
| Ⅲ      | U+2162  | ⅲ      | U+2172  |
| Ⅳ      | U+2163  | ⅳ      | U+2173  |
| Ⅴ      | U+2164  | ⅴ      | U+2174  |
| ... | ... | ... | ... |
| Ⅻ      | U+216B  | ⅻ      | U+217B  |

### 3. 処理優先順位

`applyNormalizationWithEditor`における処理順序：

1. **直接文字置換** (`replaceCharMap`)
2. **文字列置換** (`stringReplacer`)
3. **Roman numeral変換** ← **ここで処理**
4. **一般大文字→小文字変換**
5. **NFKC正規化** ← **スキップされる**

### 4. NFKC正規化がスキップされる理由

```go
// Roman numeral変換で needsChange = true に設定される
if !needsChange && p.isUpperRomanNumeral(ch) {
    lowered := unicode.ToLower(ch)
    replacement = string(lowered)
    needsChange = true  // ← これが重要
}

// NFKC正規化の条件
if !needsChange && needNFKC && !p.ignoreNormalizeSet[ch] {
    // needsChange=true のため、この処理はスキップされる
    normalized := norm.NFKC.String(charStr)
    // ...
}
```

## 技術的背景

### Unicode標準での定義

- **Roman numerals**: Unicode BlockのNumber Formsに定義
- **大文字**: U+2160-U+216F
- **小文字**: U+2170-U+217F
- **ToLower変換**: Unicode標準の大文字→小文字マッピングに準拠

### NFKC正規化との関係

```
実験結果:
Ⅲ (U+2162) → NFKC → "III" (ラテン文字)
ⅲ (U+2172) → NFKC → "iii" (ラテン文字)
```

しかし実装では：
- Roman numeral変換が先に実行される
- NFKC正規化はスキップされる
- 辞書引きは`ⅲ` (Roman numeral形式) で実行

## 辞書との整合性

### 辞書エントリの確認

Go版の動作確認結果：
```bash
echo "Ⅲ" | ./sudachi-go -a --debug
# Input dump: ⅲ
# 最終出力: Ⅲ	名詞,数詞,*,*,*,*	3	Ⅲ	サン	0	[16438]
```

これは辞書に以下のエントリが存在することを示す：
- 検索キー: `ⅲ` (小文字Roman numeral)
- 辞書形: `Ⅲ` (大文字Roman numeral)

## 結論

1. **変換処理**: `Ⅲ` → `ⅲ` (Unicode ToLower)
2. **NFKC正規化**: **実行されない**
3. **辞書引き**: Roman numeral形式（`ⅲ`）で実行
4. **設計意図**: Roman numeralの形を保持したまま処理

この設計により、Roman numeralの視覚的特性を保持しながら、大文字・小文字の統一性を確保している。

## ユーザー理解との差異

**誤解**: 「大文字→小文字→NFKC正規化→辞書引き」
**実際**: 「大文字→小文字→辞書引き（NFKC正規化なし）」

この差異は、Roman numeralの特別な処理によるものであり、一般的な文字処理とは異なる専用ロジックが適用されている。