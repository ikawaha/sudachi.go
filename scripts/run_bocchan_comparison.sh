#!/bin/bash

# 坊っちゃん全文比較テスト実行スクリプト
# Rust版ゴールデンデータとGo版の解析結果を比較

set -e

# スクリプトの場所からプロジェクトルートを決定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test/regression"

echo "=== 坊っちゃん全文比較テスト実行 ==="
echo "プロジェクトルート: $PROJECT_ROOT"
echo "テストディレクトリ: $TEST_DIR"

# プロジェクトルートに移動
cd "$PROJECT_ROOT"

# ゴールデンデータの存在確認
echo "🔍 ゴールデンデータの存在確認..."
for MODE in A B C; do
    GOLDEN_FILE="$TEST_DIR/bocchan_golden_mode_${MODE}.txt"
    if [ ! -f "$GOLDEN_FILE" ]; then
        echo "❌ ゴールデンデータが見つかりません: $GOLDEN_FILE"
        echo "    まず scripts/generate_bocchan_golden.sh を実行してください"
        exit 1
    else
        LINE_COUNT=$(wc -l < "$GOLDEN_FILE")
        echo "   ✅ モード${MODE}: $LINE_COUNT 行"
    fi
done

# 入力ファイルの存在確認
INPUT_FILE="$PROJECT_ROOT/testdata/sentences_bocchan.txt"
if [ ! -f "$INPUT_FILE" ]; then
    echo "❌ 入力ファイルが見つかりません: $INPUT_FILE"
    exit 1
fi

TOTAL_SENTENCES=$(wc -l < "$INPUT_FILE")
echo "📄 入力文数: $TOTAL_SENTENCES 文"

# Go mod tidy実行（依存関係の整理）
echo "🔧 Go依存関係の整理..."
go mod tidy

# テスト実行
echo ""
echo "🚀 坊っちゃん全文比較テスト実行開始..."
echo "   対象: $TOTAL_SENTENCES 文 × A/B/C 3モード"
echo ""

# テスト実行（詳細ログ出力）
go test -v ./test/regression -run TestBocchanFullComparison -timeout 30m

# テスト実行結果確認
TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ テスト実行完了"
else
    echo ""
    echo "❌ テスト実行でエラーが発生しました (exit code: $TEST_EXIT_CODE)"
fi

# レポートファイルの確認
REPORT_FILE="$TEST_DIR/bocchan_full_comparison_report.json"
if [ -f "$REPORT_FILE" ]; then
    echo ""
    echo "📋 詳細比較レポート生成済み: $REPORT_FILE"
    
    # JSONから簡単なサマリーを抽出表示
    if command -v jq &> /dev/null; then
        echo ""
        echo "📊 比較結果サマリー:"
        echo "   総文数: $(jq -r '.total_sentences' "$REPORT_FILE")"
        echo "   モードA一致率: $(jq -r '.match_rate.A' "$REPORT_FILE")%"
        echo "   モードB一致率: $(jq -r '.match_rate.B' "$REPORT_FILE")%"
        echo "   モードC一致率: $(jq -r '.match_rate.C' "$REPORT_FILE")%"
        echo "   実行時間: $(jq -r '.test_duration' "$REPORT_FILE")"
        
        # 不一致がある場合の表示
        MISMATCHES_A=$(jq -r '.mismatch_count.A // 0' "$REPORT_FILE")
        MISMATCHES_B=$(jq -r '.mismatch_count.B // 0' "$REPORT_FILE")
        MISMATCHES_C=$(jq -r '.mismatch_count.C // 0' "$REPORT_FILE")
        
        if [ "$MISMATCHES_A" -gt 0 ] || [ "$MISMATCHES_B" -gt 0 ] || [ "$MISMATCHES_C" -gt 0 ]; then
            echo ""
            echo "⚠️  不一致が検出されました:"
            [ "$MISMATCHES_A" -gt 0 ] && echo "   モードA: $MISMATCHES_A 件"
            [ "$MISMATCHES_B" -gt 0 ] && echo "   モードB: $MISMATCHES_B 件"
            [ "$MISMATCHES_C" -gt 0 ] && echo "   モードC: $MISMATCHES_C 件"
            echo ""
            echo "📝 詳細はレポートファイルを確認してください:"
            echo "   cat $REPORT_FILE | jq '.results[] | select(.is_match == false)'"
        fi
    else
        echo "   (jqがインストールされていないため、詳細表示をスキップ)"
    fi
else
    echo ""
    echo "⚠️  詳細比較レポートが生成されませんでした"
fi

echo ""
echo "🎯 次のステップ:"
echo "   1. レポート詳細確認: cat $REPORT_FILE | jq ."
echo "   2. 不一致分析: cat $REPORT_FILE | jq '.results[] | select(.is_match == false)'"
echo "   3. モード別統計: cat $REPORT_FILE | jq '.match_rate'"

exit $TEST_EXIT_CODE