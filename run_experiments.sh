#!/bin/bash

echo "=== Go GC 实验脚本 ==="
echo "用来跑GC测试，方便截图和收集数据"
echo ""

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "没找到Go，请先装一下Go"
    exit 1
fi

echo "Go版本: $(go version)"
echo "当前GOGC: ${GOGC:-100}"
echo ""

# 创建结果目录
mkdir -p results screenshots

show_menu() {
    echo "可以跑的实验："
    echo ""
    echo "1. basic      - 基础GC行为"
    echo "2. gogc       - GOGC参数对比"
    echo "3. pool       - 对象池效果"
    echo "4. alloc      - 分配模式对比"
    echo "5. concurrent - 并发GC测试"
    echo "6. leak       - goroutine泄漏"
    echo "7. slice      - 切片泄漏"
    echo "8. monitor    - GC监控"
    echo "9. all        - 全部跑一遍"
    echo ""
    echo "用法："
    echo "  ./run_experiments.sh [实验名]"
    echo "  比如: ./run_experiments.sh basic"
    echo ""
    echo "截图提醒："
    echo "  • 调整下终端窗口大小，显示效果更好"
    echo "  • 日志会自动存到 results/ 目录"
    echo ""
}

run_experiment() {
    local exp_name=$1

    if [ -z "$exp_name" ]; then
        show_menu
        return
    fi

    echo "开始跑实验: $exp_name"
    echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "准备截图！马上开始..."
    echo "截图文件名建议: gc_${exp_name}_$(date +%Y%m%d_%H%M%S).png"
    echo ""

    # 跑实验并保存日志
    go run main.go $exp_name 2>&1 | tee results/${exp_name}_$(date +%Y%m%d_%H%M%S).log

    echo ""
    echo "实验跑完了: $exp_name"
    echo "日志保存在: results/${exp_name}_$(date +%Y%m%d_%H%M%S).log"
    echo ""
}

# 批量跑所有实验
run_all_experiments() {
    echo "准备跑所有实验（方便连续截图）"
    echo ""

    experiments=("basic" "gogc" "pool" "alloc" "concurrent" "leak" "slice" "monitor")

    for exp in "${experiments[@]}"; do
        echo "准备跑: $exp"
        echo "按回车继续..."
        read -r

        run_experiment $exp

        echo "跑完了 - 按回车继续下一个..."
        read -r
        echo ""
    done

    echo "全部跑完了！"
    echo "截图文件在 screenshots/ 目录"
    echo "日志文件在 results/ 目录"
}

# 主逻辑
if [ $# -eq 0 ]; then
    show_menu
elif [ "$1" = "all" ]; then
    run_all_experiments
else
    run_experiment $1
fi

echo ""
echo "小提示："
echo "  • 看代码: cat main.go"
echo "  • 看日志: ls -la results/"
echo "  • 直接跑: go run main.go basic"
echo ""