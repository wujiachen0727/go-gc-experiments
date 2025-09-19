// Go GC机制深度验证实验
// Author: wujiachen | Created: 2025-09-19
// 基于真实环境的GC性能测试和分析
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		showMenu()
		return
	}

	experiment := os.Args[1]

	fmt.Println("=== Go GC 机制实验验证 ===")
	fmt.Printf("Go版本: %s\n", runtime.Version())
	fmt.Printf("操作系统: %s %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU核心: %d\n", runtime.NumCPU())
	fmt.Printf("GOGC设置: %s\n", getGOGC())
	fmt.Println()

	switch experiment {
	case "basic":
		experimentBasicGC()
	case "gogc":
		experimentGOGCComparison()
	case "pool":
		experimentObjectPool()
	case "patterns":
		experimentAllocationPatterns()
	case "concurrent":
		experimentConcurrentGC()
	case "leak":
		experimentMemoryLeak()
	case "scale":
		experimentLargeScale()
	case "all":
		runAllExperiments()
	default:
		fmt.Printf("未知实验类型: %s\n", experiment)
		showMenu()
	}
}

func showMenu() {
	fmt.Println("Go GC 实验工具")
	fmt.Println()
	fmt.Println("用法: go run main.go [实验类型]")
	fmt.Println()
	fmt.Println("实验:")
	fmt.Println("  basic     - 基础GC行为")
	fmt.Println("  gogc      - GOGC参数对比")
	fmt.Println("  pool      - 对象池对比")
	fmt.Println("  patterns  - 分配模式")
	fmt.Println("  concurrent - 并发GC")
	fmt.Println("  leak      - 内存泄漏检测")
	fmt.Println("  scale     - 大规模调优")
	fmt.Println("  all       - 全部实验")
	fmt.Println()
	fmt.Println("例子: go run main.go basic")
}

func getGOGC() string {
	gogc := os.Getenv("GOGC")
	if gogc == "" {
		return "100 (默认)"
	}
	return gogc
}

func experimentBasicGC() {
	fmt.Println("=== 基础GC行为观察 ===")

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	fmt.Printf("初始 - 堆: %s, GC: %d次\n",
		formatBytes(m1.HeapAlloc), m1.NumGC)

	const totalAllocs = 100000
	const blockSize = 4096

	fmt.Printf("分配 %d 个 %d 字节块...\n", totalAllocs, blockSize)

	data := make([][]byte, 0, totalAllocs)
	for i := 0; i < totalAllocs; i++ {
		block := make([]byte, blockSize)
		data = append(data, block)

		if i%10000 == 0 && i > 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("%d次后 - 堆: %s, GC: %d次, 暂停: %v\n",
				i, formatBytes(m.HeapAlloc), m.NumGC, time.Duration(m.PauseTotalNs))
		}
	}

	fmt.Println("手动GC...")
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	fmt.Printf("GC后 - 堆: %s, GC: %d次, 总暂停: %v\n",
		formatBytes(m2.HeapAlloc), m2.NumGC, time.Duration(m2.PauseTotalNs))

	fmt.Println("释放引用...")
	data = nil
	runtime.GC()

	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	fmt.Printf("释放后 - 堆: %s, GC: %d次\n",
		formatBytes(m3.HeapAlloc), m3.NumGC)

	fmt.Println("\n总结:")
	fmt.Printf("%-10s %-12s %-8s %-12s\n", "阶段", "堆内存", "GC次数", "暂停时间")
	fmt.Printf("%-10s %-12s %-8d %-12s\n", "初始", formatBytes(m1.HeapAlloc), m1.NumGC, time.Duration(m1.PauseTotalNs))
	fmt.Printf("%-10s %-12s %-8d %-12s\n", "分配后", formatBytes(m2.HeapAlloc), m2.NumGC, time.Duration(m2.PauseTotalNs))
	fmt.Printf("%-10s %-12s %-8d %-12s\n", "释放后", formatBytes(m3.HeapAlloc), m3.NumGC, time.Duration(m3.PauseTotalNs))
	fmt.Println()
}

func experimentGOGCComparison() {
	fmt.Println("=== GOGC参数对比 ===")

	gogcValues := []int{50, 100, 200, 400, 800}
	results := make([]GOGCResult, len(gogcValues))

	fmt.Println("测试不同GOGC值...")

	for i, gogc := range gogcValues {
		fmt.Printf("测试 GOGC=%d\n", gogc)
		debug.SetGCPercent(gogc)

		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		performMemoryWork()
		duration := time.Since(start)

		runtime.ReadMemStats(&m2)

		results[i] = GOGCResult{
			GOGC:        gogc,
			Duration:    duration,
			GCCount:     m2.NumGC - m1.NumGC,
			PauseTotal:  time.Duration(m2.PauseTotalNs - m1.PauseTotalNs),
			HeapPeak:    m2.HeapSys,
			Allocations: m2.Mallocs - m1.Mallocs,
		}

		fmt.Printf("  执行时间: %v, GC次数: %d, 总暂停: %v, 峰值堆: %s\n",
			duration.Round(time.Millisecond),
			results[i].GCCount,
			results[i].PauseTotal.Round(time.Microsecond),
			formatBytes(results[i].HeapPeak))

		time.Sleep(time.Second) // 让GC稳定
	}

	fmt.Println("\n对比结果:")
	fmt.Printf("%-6s %-10s %-6s %-10s %-10s %-10s\n",
		"GOGC", "时间", "GC次数", "暂停", "峰值堆", "分配数")

	for _, result := range results {
		fmt.Printf("%-6d %-10s %-6d %-10s %-10s %-10d\n",
			result.GOGC,
			result.Duration.Round(time.Millisecond),
			result.GCCount,
			result.PauseTotal.Round(time.Microsecond),
			formatBytes(result.HeapPeak),
			result.Allocations)
	}

	debug.SetGCPercent(100)
	fmt.Println()
}

func experimentObjectPool() {
	fmt.Println("=== 对象池 vs 频繁分配 ===")

	const iterations = 2000000 // 增加到200万次
	const bufferSize = 128     // 改为128字节小对象

	fmt.Printf("测试: %d次分配，每次%d字节\n", iterations, bufferSize)

	fmt.Println("\n1. 频繁分配...")
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()
	var sink [][]byte // 防止编译器优化
	for i := 0; i < iterations; i++ {
		data := make([]byte, bufferSize)
		// 真正使用这些内存
		for j := 0; j < len(data); j += 8 {
			data[j] = byte(i % 256)
		}
		// 每1000个保留一个，增加GC压力
		if i%1000 == 0 {
			sink = append(sink, data)
		}

		if i%500000 == 0 && i > 0 {
			fmt.Printf("   %d/%d\n", i, iterations)
		}
	}
	duration1 := time.Since(start)
	runtime.ReadMemStats(&m2)

	fmt.Printf("结果: 时间%v, GC%d次, 分配%d次, 暂停%v\n",
		duration1.Round(time.Millisecond), m2.NumGC-m1.NumGC,
		m2.Mallocs-m1.Mallocs, time.Duration(m2.PauseTotalNs-m1.PauseTotalNs).Round(time.Microsecond))

	// 清理
	sink = nil
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n2. 对象池...")
	var bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, bufferSize)
		},
	}

	var m3, m4 runtime.MemStats
	runtime.ReadMemStats(&m3)

	start = time.Now()
	var sink2 [][]byte
	for i := 0; i < iterations; i++ {
		data := bufferPool.Get().([]byte)
		// 重置并使用内存
		for j := 0; j < len(data); j += 8 {
			data[j] = byte(i % 256)
		}
		// 同样的保留策略
		if i%1000 == 0 {
			kept := make([]byte, len(data))
			copy(kept, data)
			sink2 = append(sink2, kept)
		}
		bufferPool.Put(data)

		if i%500000 == 0 && i > 0 {
			fmt.Printf("   %d/%d\n", i, iterations)
		}
	}
	duration2 := time.Since(start)
	runtime.ReadMemStats(&m4)

	fmt.Printf("结果: 时间%v, GC%d次, 分配%d次, 暂停%v\n",
		duration2.Round(time.Millisecond), m4.NumGC-m3.NumGC,
		m4.Mallocs-m3.Mallocs, time.Duration(m4.PauseTotalNs-m3.PauseTotalNs).Round(time.Microsecond))

	speedup := float64(duration1) / float64(duration2)
	allocReduction := float64(m2.Mallocs-m1.Mallocs) / float64(m4.Mallocs-m3.Mallocs+1)
	gcReduction := float64(m2.NumGC-m1.NumGC+1) / float64(m4.NumGC-m3.NumGC+1)

	fmt.Printf("\n对比: 速度提升%.2fx, 分配减少%.2fx, GC减少%.2fx\n", speedup, allocReduction, gcReduction)
	fmt.Println()
}

func experimentAllocationPatterns() {
	fmt.Println("=== 分配模式对比 ===")

	patterns := []struct {
		name string
		desc string
		fn   func() interface{}
	}{
		{"小对象", "大量小对象分配", func() interface{} {
			var result [][]byte
			for i := 0; i < 200000; i++ {
				data := make([]byte, 64)
				data[0] = byte(i % 256) // 真正使用内存
				if i%10000 == 0 {
					result = append(result, data)
				}
			}
			return result
		}},
		{"大对象", "少量大对象分配", func() interface{} {
			var result [][]byte
			for i := 0; i < 200; i++ {
				data := make([]byte, 64*1024)
				// 填充一些数据
				for j := 0; j < len(data); j += 1024 {
					data[j] = byte(i % 256)
				}
				result = append(result, data)
			}
			return result
		}},
		{"长生命周期", "长生命周期对象", func() interface{} {
			var longLived [][]byte
			for i := 0; i < 5000; i++ {
				data := make([]byte, 1024)
				data[0] = byte(i % 256)
				longLived = append(longLived, data)
			}
			time.Sleep(200 * time.Millisecond)
			return longLived
		}},
	}

	results := make([]PatternResult, len(patterns))

	for i, pattern := range patterns {
		fmt.Printf("%d. 测试: %s (%s)\n", i+1, pattern.name, pattern.desc)

		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		result := pattern.fn() // 保存结果防止优化
		duration := time.Since(start)

		runtime.ReadMemStats(&m2)

		results[i] = PatternResult{
			Name:        pattern.name,
			Duration:    duration,
			GCCount:     m2.NumGC - m1.NumGC,
			Allocations: m2.Mallocs - m1.Mallocs,
			PauseTotal:  time.Duration(m2.PauseTotalNs - m1.PauseTotalNs),
		}

		fmt.Printf("   执行时间: %v, GC次数: %d, 分配次数: %d, 暂停时间: %v\n",
			duration.Round(time.Millisecond),
			results[i].GCCount,
			results[i].Allocations,
			results[i].PauseTotal.Round(time.Microsecond))
		fmt.Println()

		_ = result // 使用结果变量
	}

	fmt.Println("对比结果:")
	fmt.Printf("%-10s %-10s %-6s %-10s %-10s\n",
		"模式", "时间", "GC次数", "分配数", "暂停")

	for _, result := range results {
		fmt.Printf("%-10s %-10s %-6d %-10d %-10s\n",
			result.Name,
			result.Duration.Round(time.Millisecond),
			result.GCCount,
			result.Allocations,
			result.PauseTotal.Round(time.Microsecond))
	}
	fmt.Println()
}

func experimentConcurrentGC() {
	fmt.Println("=== 并发GC测试 ===")

	const goroutines = 50
	const workPerGoroutine = 100000
	const blockSize = 2048

	fmt.Printf("%d个goroutine，每个%d次分配，每次%d字节\n", goroutines, workPerGoroutine, blockSize)
	fmt.Printf("总计: %d次分配，约%s内存\n", goroutines*workPerGoroutine,
		formatBytes(uint64(goroutines*workPerGoroutine*blockSize)))

	var wg sync.WaitGroup
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 共享存储，增加对象生命周期
	var sharedData sync.Map

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var localData [][]byte

			for j := 0; j < workPerGoroutine; j++ {
				data := make([]byte, blockSize)
				// 填充数据
				for k := 0; k < len(data); k += 64 {
					data[k] = byte((id + j) % 256)
				}

				// 每100个保留一个到本地
				if j%100 == 0 {
					localData = append(localData, data)
				}

				// 每1000个存到共享map，增加GC压力
				if j%1000 == 0 {
					key := fmt.Sprintf("g%d-i%d", id, j)
					sharedData.Store(key, data)
				}

				// 偶尔清理本地数据
				if len(localData) > 50 {
					localData = localData[10:]
				}
			}

			// 最后清理一些共享数据
			for k := 0; k < 10; k++ {
				key := fmt.Sprintf("g%d-i%d", id, k*1000)
				sharedData.Delete(key)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&m2)

	fmt.Printf("执行时间: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("GC次数: %d\n", m2.NumGC-m1.NumGC)
	fmt.Printf("总暂停: %v\n", time.Duration(m2.PauseTotalNs-m1.PauseTotalNs).Round(time.Microsecond))

	if m2.NumGC > m1.NumGC {
		avgPause := time.Duration((m2.PauseTotalNs - m1.PauseTotalNs) / uint64(m2.NumGC-m1.NumGC))
		maxPause := time.Duration(m2.PauseNs[(m2.NumGC+255)%256])
		fmt.Printf("平均暂停: %v\n", avgPause.Round(time.Microsecond))
		fmt.Printf("最大暂停: %v\n", maxPause.Round(time.Microsecond))
	}

	fmt.Printf("峰值堆: %s\n", formatBytes(m2.HeapSys))
	fmt.Printf("当前堆: %s\n", formatBytes(m2.HeapAlloc))

	throughput := float64(goroutines*workPerGoroutine) / duration.Seconds()
	fmt.Printf("吞吐量: %.0f ops/sec\n", throughput)

	if m2.NumGC > m1.NumGC {
		gcOverhead := float64(m2.PauseTotalNs-m1.PauseTotalNs) / float64(duration.Nanoseconds()) * 100
		fmt.Printf("GC开销: %.2f%%\n", gcOverhead)
	}
	fmt.Println()
}

// 运行所有实验
func runAllExperiments() {
	experiments := []struct {
		name string
		fn   func()
	}{
		{"基础GC行为", experimentBasicGC},
		{"GOGC参数对比", experimentGOGCComparison},
		{"对象池对比", experimentObjectPool},
		{"分配模式对比", experimentAllocationPatterns},
		{"并发GC测试", experimentConcurrentGC},
		{"内存泄漏检测", experimentMemoryLeak},
		{"大规模GC调优", experimentLargeScale},
	}

	fmt.Printf("开始运行所有 %d 个实验...\n\n", len(experiments))

	for i, exp := range experiments {
		fmt.Printf(">>> 实验 %d/%d: %s\n", i+1, len(experiments), exp.name)
		exp.fn()

		if i < len(experiments)-1 {
			fmt.Println("等待3秒后继续下一个实验...")
			time.Sleep(3 * time.Second)
			fmt.Println()
		}
	}

	fmt.Println("所有实验完成！")
}

// 辅助函数和结构体
type GOGCResult struct {
	GOGC        int
	Duration    time.Duration
	GCCount     uint32
	PauseTotal  time.Duration
	HeapPeak    uint64
	Allocations uint64
}

type PatternResult struct {
	Name        string
	Duration    time.Duration
	GCCount     uint32
	Allocations uint64
	PauseTotal  time.Duration
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func performMemoryWork() {
	var data [][]byte
	for i := 0; i < 200000; i++ {
		block := make([]byte, 2048)
		data = append(data, block)

		// 定期清理一些数据
		if i%20 == 0 && len(data) > 5000 {
			data = data[1000:]
		}
	}
}

func processData(data []byte) {
	// 简单填充数据
	for i := range data {
		data[i] = byte(i % 256)
	}
}

func experimentMemoryLeak() {
	fmt.Println("=== 内存泄漏检测 ===")

	fmt.Println("1. goroutine泄漏")
	testGoroutineLeak()

	fmt.Println("\n2. 切片容量泄漏")
	testSliceCapacityLeak()
}

func experimentLargeScale() {
	fmt.Println("=== 大规模GC调优 ===")

	fmt.Println("1. Web服务")
	testHighConcurrencyWebService()

	fmt.Println("\n2. 批处理")
	testBatchProcessingGC()

	fmt.Println("\n3. 长连接")
	testLongConnectionServiceGC()
}

// goroutine泄漏检测
func testGoroutineLeak() {
	initialGoroutines := runtime.NumGoroutine()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	fmt.Printf("初始goroutine数量: %d\n", initialGoroutines)

	// 创建会泄漏的goroutine
	for i := 0; i < 100; i++ {
		go func(id int) {
			// 模拟永远阻塞的goroutine
			ch := make(chan int)
			<-ch // 永远等待
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	leakedGoroutines := runtime.NumGoroutine()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	fmt.Printf("泄漏后goroutine数量: %d\n", leakedGoroutines)
	fmt.Printf("泄漏的goroutine: %d个\n", leakedGoroutines-initialGoroutines)
	fmt.Printf("内存增长: %s\n", formatBytes(m2.HeapAlloc-m1.HeapAlloc))

	fmt.Println("结论: 每个泄漏goroutine约2-8KB，会持续增长")
}

// 切片容量泄漏测试
func testSliceCapacityLeak() {
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 创建大切片然后只保留小部分
	fmt.Println("创建10MB大切片...")
	largeSlice := make([]byte, 10*1024*1024) // 10MB
	for i := range largeSlice {
		largeSlice[i] = byte(i % 256)
	}

	// 只保留前100字节，但整个10MB仍被引用
	smallSlice := largeSlice[:100]

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	fmt.Printf("大切片创建后内存: %s\n", formatBytes(m2.HeapAlloc))
	fmt.Printf("小切片长度: %d, 容量: %d\n", len(smallSlice), cap(smallSlice))

	// 正确的做法：复制需要的部分
	fmt.Println("修复: 复制需要的部分...")
	properSlice := make([]byte, 100)
	copy(properSlice, largeSlice[:100])
	largeSlice = nil // 释放大切片

	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	fmt.Printf("修复后内存: %s\n", formatBytes(m3.HeapAlloc))
	fmt.Printf("内存节省: %s\n", formatBytes(m2.HeapAlloc-m3.HeapAlloc))

	fmt.Println("结论: 切片容量决定内存占用，用copy()避免泄漏")

	_ = properSlice // 使用变量避免编译器优化
}

// 高并发Web服务GC调优
func testHighConcurrencyWebService() {
	const requests = 10000
	const concurrency = 50

	fmt.Printf("模拟处理%d个请求，并发度%d\n", requests, concurrency)

	// 测试不同GOGC值下的表现
	gogcValues := []int{50, 100, 200}

	for _, gogc := range gogcValues {
		debug.SetGCPercent(gogc)

		var wg sync.WaitGroup
		start := time.Now()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// 创建请求处理goroutine
		requestChan := make(chan int, requests)
		for i := 0; i < requests; i++ {
			requestChan <- i
		}
		close(requestChan)

		// 启动工作goroutine
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for reqID := range requestChan {
					// 模拟请求处理：分配内存、处理、释放
					data := make([]byte, 4096)
					processData(data)

					// 模拟一些计算
					_ = reqID * 2
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		fmt.Printf("  GOGC=%d: 处理%d请求耗时%v, GC次数%d, 平均延迟%v\n",
			gogc, requests, duration.Round(time.Millisecond),
			m2.NumGC-m1.NumGC, (duration / time.Duration(requests)).Round(time.Microsecond))
	}

	debug.SetGCPercent(100) // 恢复默认值
	fmt.Println("结论: Web服务GOGC=100-200较好")
}

// 批处理任务GC调优
func testBatchProcessingGC() {
	const batchSize = 50000
	const itemSize = 2048

	fmt.Printf("模拟批处理%d个项目，每项%d字节\n", batchSize, itemSize)

	gogcValues := []int{100, 400, 800}

	for _, gogc := range gogcValues {
		debug.SetGCPercent(gogc)

		start := time.Now()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// 批量处理数据
		batch := make([][]byte, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			data := make([]byte, itemSize)
			processData(data)
			batch = append(batch, data)

			// 每处理5000个项目检查一次
			if i%5000 == 0 && i > 0 {
				// 模拟批量写入，然后清理
				batch = batch[:0]
				if i%10000 == 0 {
					runtime.GC()
				}
			}
		}

		duration := time.Since(start)
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		fmt.Printf("  GOGC=%d: 处理%d项耗时%v, GC次数%d, 吞吐量%.0f项/秒\n",
			gogc, batchSize, duration.Round(time.Millisecond),
			m2.NumGC-m1.NumGC, float64(batchSize)/duration.Seconds())
	}

	debug.SetGCPercent(100)
	fmt.Println("结论: 批处理GOGC=400-800，优先吞吐量")
}

// 长连接服务GC调优
func testLongConnectionServiceGC() {
	const connections = 500
	const messagesPerConn = 200

	fmt.Printf("模拟%d个长连接，每连接处理%d个消息\n", connections, messagesPerConn)

	var wg sync.WaitGroup
	start := time.Now()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 创建连接池
	connectionPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}

	// 模拟长连接处理
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			// 每个连接处理多个消息
			for j := 0; j < messagesPerConn; j++ {
				// 从池中获取缓冲区
				buffer := connectionPool.Get().([]byte)

				// 模拟消息处理
				processData(buffer)

				// 归还到池中
				connectionPool.Put(buffer)

				// 模拟消息间隔
				time.Sleep(time.Microsecond * 10)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	totalMessages := connections * messagesPerConn
	fmt.Printf("长连接服务: %d连接×%d消息, 耗时%v\n", connections, messagesPerConn, duration.Round(time.Millisecond))
	fmt.Printf("GC次数: %d, 消息吞吐量: %.0f消息/秒\n",
		m2.NumGC-m1.NumGC, float64(totalMessages)/duration.Seconds())
	fmt.Printf("平均每连接内存: %s\n",
		formatBytes((m2.HeapAlloc-m1.HeapAlloc)/uint64(connections)))

	fmt.Println("结论: 长连接用对象池减少GC压力")
}
