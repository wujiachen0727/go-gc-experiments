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

	switch os.Args[1] {
	case "basic":
		experimentBasicGC()
	case "gogc":
		experimentGOGCComparison()
	case "pool":
		experimentObjectPool()
	case "alloc":
		experimentAllocationPatterns()
	case "concurrent":
		experimentConcurrentGC()
	case "leak":
		experimentGoroutineLeak()
	case "slice":
		experimentSliceLeak()
	case "monitor":
		monitorGCPerformance()
	default:
		showMenu()
	}
}

func showMenu() {
	fmt.Println("Go GC机制验证实验")
	fmt.Println("=================")
	fmt.Println("核心实验:")
	fmt.Println("  basic     - 基础GC行为观察")
	fmt.Println("  gogc      - GOGC参数对比")
	fmt.Println("  pool      - 对象池效果验证")
	fmt.Println("进阶实验:")
	fmt.Println("  alloc     - 内存分配模式对比")
	fmt.Println("  concurrent- 并发场景GC测试")
	fmt.Println("  leak      - Goroutine泄漏检测")
	fmt.Println("  slice     - 切片容量泄漏检测")
	fmt.Println("  monitor   - GC性能监控")
	fmt.Println()
	fmt.Println("使用方法: go run main.go <实验名称>")
	fmt.Println("例如: go run main.go basic")
}

func experimentBasicGC() {
	fmt.Println("=== 基础GC行为观察 ===")

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	fmt.Printf("初始状态 - 堆内存: %s, GC次数: %d\n",
		formatBytes(m1.HeapAlloc), m1.NumGC)

	const totalAllocs = 100000
	const blockSize = 4096

	data := make([][]byte, 0, totalAllocs)
	for i := 0; i < totalAllocs; i++ {
		block := make([]byte, blockSize)
		data = append(data, block)

		if i%10000 == 0 && i > 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("分配%d次后 - 堆内存: %s, GC次数: %d\n",
				i, formatBytes(m.HeapAlloc), m.NumGC)
		}
	}

	// 释放引用，观察GC回收效果
	data = nil
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	fmt.Printf("手动GC后 - 堆内存: %s, GC次数: %d\n",
		formatBytes(m2.HeapAlloc), m2.NumGC)
}

func experimentGOGCComparison() {
	fmt.Println("=== GOGC参数对比测试 ===")

	gogcValues := []int{50, 100, 200, 400}

	for _, gogc := range gogcValues {
		fmt.Printf("\n--- 测试GOGC=%d ---\n", gogc)
		debug.SetGCPercent(gogc)

		// 先手动GC一次，清理之前的影响
		runtime.GC()
		runtime.GC()

		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()

		// 增加内存分配量，确保能触发GC
		const iterations = 200000 // 增加到20万次
		const blockSize = 8192    // 增加到8KB

		// 用切片保持一部分对象存活，增加GC压力
		keepAlive := make([][]byte, 0, iterations/10)

		for i := 0; i < iterations; i++ {
			data := make([]byte, blockSize)
			processData(data)

			// 每10个对象保留1个，增加内存压力
			if i%10 == 0 {
				keepAlive = append(keepAlive, data)
			}

			// 每1万次分配后检查一下内存，让GC有机会运行
			if i%10000 == 0 && i > 0 {
				runtime.Gosched() // 让出CPU时间
			}
		}

		duration := time.Since(start)

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		gcCount := m2.NumGC - m1.NumGC
		totalAlloc := m2.TotalAlloc - m1.TotalAlloc

		fmt.Printf("GOGC=%d: 耗时%v, GC次数%d, 总分配%s, 当前堆%s\n",
			gogc, duration, gcCount, formatBytes(totalAlloc), formatBytes(m2.HeapAlloc))

		// 清理keepAlive，避免影响下次测试
		keepAlive = nil
		runtime.GC()
	}

	// 恢复默认值
	debug.SetGCPercent(100)
}

func experimentObjectPool() {
	fmt.Println("=== 对象池效果对比 ===")

	const iterations = 2000000 // 增加到200万次
	const bufferSize = 4096    // 增加到4KB

	// 先清理环境
	runtime.GC()
	runtime.GC()

	// 测试频繁分配
	fmt.Println("\n--- 频繁分配测试 ---")
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()
	// 保持一些对象存活，增加GC压力
	keepAlive := make([][]byte, 0, iterations/100)

	for i := 0; i < iterations; i++ {
		data := make([]byte, bufferSize)
		processData(data)

		// 每100个保留1个
		if i%100 == 0 {
			keepAlive = append(keepAlive, data)
		}
	}
	duration1 := time.Since(start)

	runtime.ReadMemStats(&m2)
	gcCount1 := m2.NumGC - m1.NumGC

	fmt.Printf("频繁分配: 耗时%v, GC次数%d, 分配%d次\n",
		duration1, gcCount1, iterations)

	// 清理
	keepAlive = nil
	runtime.GC()

	// 测试对象池
	fmt.Println("\n--- 对象池测试 ---")

	bufferPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, bufferSize)
		},
	}

	runtime.ReadMemStats(&m1)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		data := bufferPool.Get().([]byte)
		processData(data)
		bufferPool.Put(data)
	}
	duration2 := time.Since(start)

	runtime.ReadMemStats(&m2)
	gcCount2 := m2.NumGC - m1.NumGC

	fmt.Printf("对象池: 耗时%v, GC次数%d, 重用率高\n",
		duration2, gcCount2)

	fmt.Printf("\n性能对比:\n")
	if gcCount2 > 0 {
		fmt.Printf("  GC次数减少: %.1fx\n", float64(gcCount1)/float64(gcCount2))
	} else if gcCount1 > 0 {
		fmt.Printf("  GC次数减少: %dx (对象池几乎无GC)\n", gcCount1)
	}
	fmt.Printf("  时间对比: %.1fx\n", float64(duration1)/float64(duration2))
}

func experimentAllocationPatterns() {
	fmt.Println("=== 内存分配模式对比 ===")

	const itemCount = 500000 // 增加到50万

	// 先清理环境
	runtime.GC()
	runtime.GC()

	// 测试小对象频繁分配
	fmt.Println("\n--- 小对象频繁分配 ---")
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()
	// 保持一些对象存活
	keepAlive1 := make([][]byte, 0, itemCount/50)

	for i := 0; i < itemCount; i++ {
		data := make([]byte, 1024) // 增加到1KB
		processData(data)

		// 每50个保留1个
		if i%50 == 0 {
			keepAlive1 = append(keepAlive1, data)
		}
	}
	smallObjTime := time.Since(start)

	runtime.ReadMemStats(&m2)
	smallObjGC := m2.NumGC - m1.NumGC

	// 清理
	keepAlive1 = nil
	runtime.GC()

	// 测试大对象少量分配
	fmt.Println("\n--- 大对象少量分配 ---")
	runtime.ReadMemStats(&m1)

	start = time.Now()
	keepAlive2 := make([][]byte, 0, itemCount/1000)

	for i := 0; i < itemCount/100; i++ { // 减少次数但增加大小
		data := make([]byte, 128*1024) // 128KB大对象
		processData(data)

		// 每10个保留1个
		if i%10 == 0 {
			keepAlive2 = append(keepAlive2, data)
		}
	}
	largeObjTime := time.Since(start)

	runtime.ReadMemStats(&m2)
	largeObjGC := m2.NumGC - m1.NumGC

	// 清理
	keepAlive2 = nil
	runtime.GC()

	// 测试预分配策略
	fmt.Println("\n--- 预分配策略 ---")
	runtime.ReadMemStats(&m1)

	start = time.Now()
	buffer := make([]byte, 1024*1024) // 预分配1MB缓冲区
	for i := 0; i < itemCount; i++ {
		// 重用缓冲区的一部分
		data := buffer[:1024]
		processData(data)
	}
	preAllocTime := time.Since(start)

	runtime.ReadMemStats(&m2)
	preAllocGC := m2.NumGC - m1.NumGC

	fmt.Printf("\n结果对比:\n")
	fmt.Printf("小对象频繁: 耗时%v, GC次数%d\n", smallObjTime, smallObjGC)
	fmt.Printf("大对象少量: 耗时%v, GC次数%d\n", largeObjTime, largeObjGC)
	fmt.Printf("预分配策略: 耗时%v, GC次数%d\n", preAllocTime, preAllocGC)
}

func experimentConcurrentGC() {
	fmt.Println("=== 并发场景GC测试 ===")

	concurrencyLevels := []int{1, 10, 50, 100}
	const totalWork = 1000000 // 增加到100万次

	for _, level := range concurrencyLevels {
		fmt.Printf("\n--- %d个goroutine并发测试 ---\n", level)

		// 清理环境
		runtime.GC()
		runtime.GC()

		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		var wg sync.WaitGroup

		workPerGoroutine := totalWork / level

		for i := 0; i < level; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// 每个goroutine保持一些对象存活
				keepAlive := make([][]byte, 0, workPerGoroutine/100)

				for j := 0; j < workPerGoroutine; j++ {
					data := make([]byte, 2048) // 增加到2KB
					processData(data)

					// 每100个保留1个
					if j%100 == 0 {
						keepAlive = append(keepAlive, data)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		runtime.ReadMemStats(&m2)
		gcCount := m2.NumGC - m1.NumGC

		if gcCount > 0 {
			fmt.Printf("%d并发: 耗时%v, GC次数%d, 平均每次GC间隔%d次分配\n",
				level, duration, gcCount, totalWork/int(gcCount))
		} else {
			fmt.Printf("%d并发: 耗时%v, GC次数%d, 未触发GC\n",
				level, duration, gcCount)
		}
	}
}

func experimentGoroutineLeak() {
	fmt.Println("=== Goroutine泄漏检测 ===")

	initialGoroutines := runtime.NumGoroutine()
	fmt.Printf("初始goroutine数量: %d\n", initialGoroutines)

	// 创建会泄漏的goroutine
	for i := 0; i < 100; i++ {
		go func(id int) {
			ch := make(chan int)
			<-ch // 永远等待，造成泄漏
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	leakedGoroutines := runtime.NumGoroutine()
	fmt.Printf("泄漏后goroutine数量: %d\n", leakedGoroutines)
	fmt.Printf("泄漏的goroutine: %d个\n", leakedGoroutines-initialGoroutines)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("当前堆内存: %s\n", formatBytes(m.HeapAlloc))
}

func experimentSliceLeak() {
	fmt.Println("=== 切片容量泄漏检测 ===")

	var m1, m2, m3 runtime.MemStats

	fmt.Println("创建10MB大切片...")
	runtime.ReadMemStats(&m1)

	largeSlice := make([]byte, 10*1024*1024)
	runtime.ReadMemStats(&m2)
	fmt.Printf("大切片创建后内存: %.1f MB\n", float64(m2.HeapAlloc)/1024/1024)

	// 问题：小切片仍然持有大切片的底层数组
	smallSlice := largeSlice[:100]
	fmt.Printf("小切片长度: %d, 容量: %d\n", len(smallSlice), cap(smallSlice))

	// 解决方案：复制需要的部分
	fmt.Println("修复: 复制需要的部分...")
	fixedSlice := make([]byte, 100)
	copy(fixedSlice, largeSlice[:100])
	largeSlice = nil
	smallSlice = nil

	runtime.GC()
	runtime.ReadMemStats(&m3)
	fmt.Printf("修复后内存: %.1f MB\n", float64(m3.HeapAlloc)/1024/1024)
	fmt.Printf("内存节省: %.1f MB\n", float64(m2.HeapAlloc-m3.HeapAlloc)/1024/1024)
}

func monitorGCPerformance() {
	fmt.Println("=== GC性能监控 ===")
	fmt.Println("监控5秒钟的GC活动...")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastGC uint32
	var lastPauseTotal uint64

	// 启动一些后台工作来触发GC
	go func() {
		for {
			data := make([]byte, 1024*1024) // 1MB
			processData(data)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			if m.NumGC > lastGC {
				newPauseTotal := m.PauseTotalNs - lastPauseTotal
				newGCCount := m.NumGC - lastGC

				if newGCCount > 0 {
					avgPause := time.Duration(newPauseTotal / uint64(newGCCount))
					fmt.Printf("[GC] 堆: %dMB, 新增GC: %d次, 平均暂停: %v\n",
						m.HeapAlloc/1024/1024, newGCCount, avgPause)
				}

				lastGC = m.NumGC
				lastPauseTotal = m.PauseTotalNs
			}

		case <-timeout:
			fmt.Println("监控结束")
			return
		}
	}
}

// 辅助函数
func processData(data []byte) {
	// 模拟数据处理
	for i := range data {
		data[i] = byte(i % 256)
	}
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
