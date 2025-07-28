#!/bin/bash

# 回帰テスト実行スクリプト
# 使用方法: ./scripts/run_regression_tests.sh [--generate-baseline]

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "=== Sudachi Go 回帰テスト実行 ==="
echo "プロジェクトルート: $PROJECT_ROOT"
echo

# 辞書ファイルの存在確認
if [ ! -f "resources/system.dic" ]; then
    echo "⚠️  辞書ファイルが見つかりません。辞書をダウンロードします..."
    ./scripts/fetch_dictionary.sh
fi

# ベースライン生成モードかチェック
if [ "$1" = "--generate-baseline" ]; then
    echo "📝 ベースラインデータを生成します..."
    cd test/regression
    
    # 既存のベースラインをバックアップ
    for baseline_file in baseline.json bocchan_baseline.json; do
        if [ -f "$baseline_file" ]; then
            cp "$baseline_file" "${baseline_file}.backup.$(date +%Y%m%d_%H%M%S)"
            echo "既存の${baseline_file}をバックアップしました"
        fi
    done
    
    # TestGenerateBaselineのスキップを解除して実行
    echo "基本形態素解析のベースラインを生成中..."
    go test -v -run TestGenerateBaseline -args -generate
    
    echo "坊っちゃん文章のベースラインを生成中..."
    go test -v -run TestGenerateBocchanBaseline -args -generate
    
    # 生成結果の確認
    baseline_generated=false
    if [ -f "baseline.json" ]; then
        echo "✅ 基本ベースラインが正常に生成されました"
        echo "📊 基本ベースラインの内容:"
        echo "   - $(jq '.test_cases | length' baseline.json) 個のテストケース"
        echo "   - ファイルサイズ: $(wc -c < baseline.json) bytes"
        baseline_generated=true
    fi
    
    if [ -f "bocchan_baseline.json" ]; then
        echo "✅ 坊っちゃんベースラインが正常に生成されました"
        echo "📊 坊っちゃんベースラインの内容:"
        echo "   - $(jq '.test_sentences | length' bocchan_baseline.json) 個の文章"
        echo "   - ファイルサイズ: $(wc -c < bocchan_baseline.json) bytes"
        baseline_generated=true
    fi
    
    if [ "$baseline_generated" = false ]; then
        echo "❌ ベースライン生成に失敗しました"
        exit 1
    fi
    
    cd "$PROJECT_ROOT"
    echo
fi

echo "🧪 回帰テストを実行します..."

# 回帰テストの実行
cd test/regression
if go test -v .; then
    echo
    echo "✅ 全ての回帰テストが成功しました"
    
    # テスト結果のサマリー
    echo
    echo "📊 テスト結果サマリー:"
    echo "   - 基本形態素解析テスト: 成功"
    echo "   - モード別一貫性テスト: 成功"
    
    if [ -f "baseline.json" ]; then
        echo "   - 基本ベースライン比較: 成功"
        echo "   - 基本ベースラインファイル: $(jq '.test_cases | length' baseline.json) 個のテストケース"
    fi
    
    if [ -f "bocchan_baseline.json" ]; then
        echo "   - 坊っちゃんベースライン比較: 成功"
        echo "   - 坊っちゃんベースラインファイル: $(jq '.test_sentences | length' bocchan_baseline.json) 個の文章"
    fi
    
else
    echo
    echo "❌ 回帰テストが失敗しました"
    echo
    echo "🔍 トラブルシューティング:"
    echo "   1. 最近の変更を確認してください"
    echo "   2. 意図的な変更の場合は --generate-baseline でベースラインを更新してください"
    echo "   3. 特に「すもも問題」の解析結果を確認してください"
    echo
    exit 1
fi

cd "$PROJECT_ROOT"

echo
echo "🎉 回帰テストが完了しました"

# 次のステップの案内
echo
echo "📋 次のステップ:"
echo "   - 開発を続ける場合: 定期的にこのスクリプトを実行してください"
echo "   - 新機能追加時: test/regression/ に追加テストを作成してください"
echo "   - 破壊的変更時: --generate-baseline でベースラインを更新してください"