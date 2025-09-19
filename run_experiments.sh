#!/bin/bash

echo "=== Go GC 实验工具 ==="
echo "适合截图展示和数据收集的GC性能测试"
echo ""

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

echo "Go版本: $(go version)"
echo "当前GOGC设置: ${GOGC:-100}"
echo ""

# 创建结果目录
mkdir -p results screenshots

show_menu() {
    echo "可用的实验项目："
    echo ""
    echo "1. basic      - 基础GC行为观察"
    echo "2. gogc       - GOGC参数影响对比"  
    echo "3. pool       - 对象池vs频繁分配"
    echo "4. patterns   - 分配模式对比"
    echo "5. concurrent - 并发场景GC表现"
    echo "6. all        - 运行所有实验"
    echo ""
    echo "使用方法："
    echo "  ./run_experiments.sh [实验名称]"
    echo "  例如: ./run_experiments.sh basic"
    echo ""
    echo "截图建议："
    echo "  • 调整终端窗口大小获得最佳显示效果"
    echo "  • 每个实验运行后截图保存到 screenshots/ 目录"
    echo "  • 实验日志会自动保存到 results/ 目录"
    echo ""
}

run_experiment() {
    local exp_name=$1
    
    if [ -z "$exp_name" ]; then
        show_menu
        return
    fi
    
    echo "开始执行实验: $exp_name"
    echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "准备截图！实验即将开始..."
    echo "建议截图文件名: gc_${exp_name}_$(date +%Y%m%d_%H%M%S).png"
    echo ""
    
    # 运行实验并保存日志
    go run main.go $exp_name 2>&1 | tee results/${exp_name}_$(date +%Y%m%d_%H%M%S).log
    
    echo ""
    echo "实验完成: $exp_name"
    echo "日志已保存到: results/${exp_name}_$(date +%Y%m%d_%H%M%S).log"
    echo ""
}

# 批量运行所有实验
run_all_experiments() {
    echo "批量运行所有实验（适合连续截图）"
    echo ""
    
    experiments=("basic" "gogc" "pool" "patterns" "concurrent")
    
    for exp in "${experiments[@]}"; do
        echo "准备运行实验: $exp"
        echo "按回车键继续..."
        read -r
        
        run_experiment $exp
        
        echo "实验间隔 - 按回车键继续下一个实验..."
        read -r
        echo ""
    done
    
    echo "所有实验完成！"
    echo "请检查 screenshots/ 目录中的截图文件"
    echo "请检查 results/ 目录中的日志文件"
}

# 主逻辑
if [ $# -eq 0 ]; then
    show_menu
elif [ "$1" = "batch" ]; then
    run_all_experiments
else
    run_experiment $1
fi

echo ""
echo "提示："
echo "  • 查看实验代码: cat main.go"
echo "  • 查看实验日志: ls -la results/"
echo "  • 基准测试: go test -bench=. -benchmem"
echo ""