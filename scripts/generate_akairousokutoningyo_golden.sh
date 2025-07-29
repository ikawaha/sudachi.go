#!/bin/bash

# 赤いろうそくと人魚全文のゴールデンデータ生成スクリプト
# Rust版Sudachiを使用してA/B/Cモード別の解析結果を生成

set -e

# スクリプトの場所からプロジェクトルートを決定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RUST_DIR="$PROJECT_ROOT/sudachi.rs"
TEST_DIR="$PROJECT_ROOT/test/regression"

echo "=== 赤いろうそくと人魚全文ゴールデンデータ生成 ==="
echo "プロジェクトルート: $PROJECT_ROOT"
echo "Rust版ディレクトリ: $RUST_DIR"
echo "テストディレクトリ: $TEST_DIR"

# Rust版バイナリの存在確認
RUST_BINARY="$RUST_DIR/target/release/sudachi"
if [ ! -f "$RUST_BINARY" ]; then
    echo "❌ Rust版バイナリが見つかりません: $RUST_BINARY"
    echo "    次のコマンドでビルドしてください:"
    echo "    cd sudachi.rs && cargo build --release"
    exit 1
fi

# 入力ファイルの存在確認
INPUT_FILE="$PROJECT_ROOT/testdata/sentences_akairousokutoningyo.txt"
if [ ! -f "$INPUT_FILE" ]; then
    echo "❌ 入力ファイルが見つかりません: $INPUT_FILE"
    exit 1
fi

# 設定ファイルの存在確認
CONFIG_FILE="$RUST_DIR/resources/sudachi.json"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ 設定ファイルが見つかりません: $CONFIG_FILE"
    exit 1
fi

# テストディレクトリ作成
mkdir -p "$TEST_DIR"

# 入力文数確認
TOTAL_LINES=$(wc -l < "$INPUT_FILE")
echo "📄 入力文数: $TOTAL_LINES 文"

# A/B/Cモードそれぞれでゴールデンデータ生成
for MODE in A B C; do
    OUTPUT_FILE="$TEST_DIR/akairousokutoningyo_golden_mode_${MODE}.txt"
    
    echo "🔄 モード${MODE}の解析を実行中..."
    echo "   入力: $INPUT_FILE"
    echo "   出力: $OUTPUT_FILE"
    echo "   設定: $CONFIG_FILE"
    
    # Rust版で解析実行（--allフラグで全フィールドを出力、--split-sentences noで文境界分割を無効化）
    cd "$RUST_DIR"
    cat "$INPUT_FILE" | ./target/release/sudachi -m "$MODE" -r resources/sudachi.json --all --split-sentences no > "$OUTPUT_FILE"
    
    # 結果確認
    if [ -f "$OUTPUT_FILE" ]; then
        OUTPUT_LINES=$(wc -l < "$OUTPUT_FILE")
        echo "   ✅ モード${MODE}完了: $OUTPUT_LINES 行生成"
    else
        echo "   ❌ モード${MODE}失敗: 出力ファイルが生成されませんでした"
        exit 1
    fi
done

# 生成結果サマリー
echo ""
echo "📊 ゴールデンデータ生成完了:"
for MODE in A B C; do
    OUTPUT_FILE="$TEST_DIR/akairousokutoningyo_golden_mode_${MODE}.txt"
    if [ -f "$OUTPUT_FILE" ]; then
        FILE_SIZE=$(wc -c < "$OUTPUT_FILE")
        LINE_COUNT=$(wc -l < "$OUTPUT_FILE")
        echo "   モード${MODE}: $LINE_COUNT 行, $(($FILE_SIZE / 1024)) KB"
    fi
done

echo ""
echo "🎉 全モードのゴールデンデータ生成が完了しました"
echo ""
echo "📋 次のステップ:"
echo "   1. Go版テストを実行: go test -v ./test/regression -run TestAkairousokutoningyoFullComparison"
echo "   2. 比較結果確認: cat test/regression/akairousokutoningyo_full_comparison_report.json"