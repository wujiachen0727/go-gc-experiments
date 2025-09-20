# Go GC 实验代码

这是配合博客文章《Go GC机制深度解析：从标记清除到三色标记》的实验代码。

## 快速开始

```bash
# 直接跑实验
go run main.go basic

# 或者用脚本（方便截图）
chmod +x run_experiments.sh
./run_experiments.sh basic
```

## 实验列表

| 命令 | 说明 | 对应文章章节 |
|------|------|-------------|
| `basic` | 基础GC行为观察 | 实际测试：基础GC行为观察 |
| `gogc` | GOGC参数对比 | GOGC参数的实际影响 |
| `pool` | 对象池效果验证 | 对象池的效果验证 |
| `alloc` | 内存分配模式对比 | 内存分配模式对比 |
| `concurrent` | 并发场景GC测试 | 并发场景下的GC表现 |
| `leak` | Goroutine泄漏检测 | goroutine泄漏检测 |
| `slice` | 切片容量泄漏检测 | 切片容量泄漏检测 |
| `monitor` | GC性能监控 | 实时GC监控 |

## 使用脚本

脚本会自动保存日志和提醒截图：

```bash
# 跑单个实验
./run_experiments.sh basic

# 跑所有实验（适合连续截图）
./run_experiments.sh all
```

## 文件说明

- `main.go` - 主要实验代码
- `run_experiments.sh` - 实验脚本，方便截图
- `results/` - 实验日志保存目录

## 环境要求

- Go 1.18+
- 建议在Linux/macOS下运行
- 终端窗口调整到合适大小以便截图

## 注意事项

1. 实验会分配大量内存，请确保系统内存充足
2. 某些实验（如leak）会故意创建泄漏，属于正常现象
3. 不同机器的结果可能有差异，这很正常

## 相关链接

- 博客文章：[Go GC机制深度解析](https://wujiachen0727.github.io/posts/go-gc机制深度解析从标记清除到三色标记/)
- 作者博客：https://wujiachen0727.github.io/