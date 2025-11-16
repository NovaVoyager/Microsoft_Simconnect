package main

import (
	"fmt"
	"log"
	"time"

	main2 "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 这个示例展示更高级的 MobiFlight WASM 使用场景

func main() {
	fmt.Println("=== MobiFlight WASM 高级示例 ===\n")

	// 选择要运行的示例
	examples := []struct {
		name string
		fn   func() error
	}{
		{"监控飞机参数", monitorAircraftParameters},
		{"自动驾驶控制", autopilotControl},
		{"飞行数据记录器", flightDataLogger},
	}

	for i, ex := range examples {
		fmt.Printf("%d. %s\n", i+1, ex.name)
	}

	fmt.Print("\n请选择示例编号 (1-3): ")
	var choice int
	fmt.Scanf("%d", &choice)

	if choice < 1 || choice > len(examples) {
		log.Fatal("无效的选择")
	}

	if err := examples[choice-1].fn(); err != nil {
		log.Fatalf("示例执行失败: %v", err)
	}
}

// 示例 1: 监控飞机参数
func monitorAircraftParameters() error {
	fmt.Println("\n=== 监控飞机参数 ===")

	// 初始化客户端
	sc, mf, err := main2.SetupMobiFlightClient("Aircraft Monitor")
	if err != nil {
		return err
	}
	defer sc.Close()

	fmt.Println("✓ 连接成功，开始监控...\n")

	// 要监控的参数列表
	parameters := []string{
		"XMLVAR_Autopilot_Altitude",
		"XMLVAR_Airspeed_Mode",
		"XMLVAR_Switch_AP_MSL_Mode",
	}

	// 持续监控循环
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			fmt.Println("\n监控结束")
			return nil

		case <-ticker.C:
			fmt.Printf("[%s] 飞机参数:\n", time.Now().Format("15:04:05"))

			// 批量请求所有参数
			for _, param := range parameters {
				if err := mf.GetLVar(param); err != nil {
					log.Printf("  获取 %s 失败: %v", param, err)
					continue
				}

				// 尝试接收响应
				// 注意: 实际应用中需要更复杂的消息处理逻辑
				time.Sleep(50 * time.Millisecond)

				response, err := mf.ReceiveResponse()
				if err != nil {
					fmt.Printf("  %s: (错误: %v)\n", param, err)
				} else if response != nil {
					value, _ := main2.ParseResponseFloat(response)
					fmt.Printf("  %s: %.2f\n", param, value)
				} else {
					fmt.Printf("  %s: (等待响应...)\n", param)
				}
			}
			fmt.Println()
		}
	}
}

// 示例 2: 自动驾驶控制
func autopilotControl() error {
	fmt.Println("\n=== 自动驾驶控制示例 ===")

	// 初始化客户端
	sc, mf, err := main2.SetupMobiFlightClient("Autopilot Controller")
	if err != nil {
		return err
	}
	defer sc.Close()

	fmt.Println("✓ 连接成功\n")

	// 定义自动驾驶配置
	type AutopilotConfig struct {
		AltitudeMode int
		TargetAlt    float64
		SpeedMode    int
		TargetSpeed  float64
	}

	config := AutopilotConfig{
		AltitudeMode: 1,
		TargetAlt:    10000.0, // 10,000 英尺
		SpeedMode:    2,
		TargetSpeed:  250.0, // 250 节
	}

	fmt.Printf("配置自动驾驶:\n")
	fmt.Printf("  - 高度模式: %d\n", config.AltitudeMode)
	fmt.Printf("  - 目标高度: %.0f 英尺\n", config.TargetAlt)
	fmt.Printf("  - 速度模式: %d\n", config.SpeedMode)
	fmt.Printf("  - 目标速度: %.0f 节\n\n", config.TargetSpeed)

	// 使用批量操作设置所有参数
	batch := main2.NewLVarBatch(mf)
	batch.Add("XMLVAR_Switch_AP_MSL_Mode", float64(config.AltitudeMode))
	batch.Add("XMLVAR_Autopilot_Altitude", config.TargetAlt)
	batch.Add("XMLVAR_Airspeed_Mode", float64(config.SpeedMode))

	fmt.Println("发送自动驾驶配置...")
	if err := batch.Execute(); err != nil {
		return fmt.Errorf("配置失败: %v", err)
	}

	fmt.Println("✓ 自动驾驶配置成功\n")

	// 等待一会儿，然后验证设置
	time.Sleep(1 * time.Second)

	fmt.Println("验证设置...")
	if err := mf.GetLVar("XMLVAR_Autopilot_Altitude"); err != nil {
		return err
	}

	time.Sleep(200 * time.Millisecond)

	response, err := mf.ReceiveResponse()
	if err != nil {
		fmt.Printf("接收响应时出错: %v\n", err)
	} else if response != nil {
		value, _ := main2.ParseResponseFloat(response)
		fmt.Printf("当前目标高度: %.0f 英尺\n", value)
	} else {
		fmt.Println("暂时没有收到响应")
	}

	fmt.Println("\n✓ 自动驾驶控制完成")
	return nil
}

// 示例 3: 飞行数据记录器
func flightDataLogger() error {
	fmt.Println("\n=== 飞行数据记录器 ===")

	// 初始化客户端
	sc, mf, err := main2.SetupMobiFlightClient("Flight Data Logger")
	if err != nil {
		return err
	}
	defer sc.Close()

	fmt.Println("✓ 连接成功\n")
	fmt.Println("开始记录飞行数据（持续 20 秒）...\n")

	// 数据记录结构
	type FlightDataPoint struct {
		Timestamp time.Time
		Altitude  float64
		Speed     float64
		Mode      float64
	}

	var flightLog []FlightDataPoint

	// 记录循环
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(20 * time.Second)
	startTime := time.Now()

	for {
		select {
		case <-timeout:
			// 记录结束，输出统计信息
			fmt.Println("\n记录结束，统计信息:")
			fmt.Printf("  - 记录点数: %d\n", len(flightLog))
			fmt.Printf("  - 记录时长: %.1f 秒\n", time.Since(startTime).Seconds())

			if len(flightLog) > 0 {
				// 计算平均值
				var avgAlt, avgSpeed float64
				for _, data := range flightLog {
					avgAlt += data.Altitude
					avgSpeed += data.Speed
				}
				avgAlt /= float64(len(flightLog))
				avgSpeed /= float64(len(flightLog))

				fmt.Printf("  - 平均高度: %.2f\n", avgAlt)
				fmt.Printf("  - 平均速度: %.2f\n", avgSpeed)

				// 显示最后 5 条记录
				fmt.Println("\n最后 5 条记录:")
				start := len(flightLog) - 5
				if start < 0 {
					start = 0
				}
				for i := start; i < len(flightLog); i++ {
					data := flightLog[i]
					fmt.Printf("  [%s] 高度: %.2f, 速度: %.2f, 模式: %.0f\n",
						data.Timestamp.Format("15:04:05"),
						data.Altitude,
						data.Speed,
						data.Mode,
					)
				}
			}

			return nil

		case <-ticker.C:
			// 使用 RPN 代码一次性读取多个参数
			// 注意: 这是一个简化示例，实际应用中需要更复杂的响应处理
			rpnCode := "(L:XMLVAR_Autopilot_Altitude) (L:XMLVAR_Airspeed_Mode)"

			if err := mf.ExecuteCode(rpnCode); err != nil {
				log.Printf("执行 RPN 代码失败: %v", err)
				continue
			}

			// 模拟数据点（实际应用中需要从响应中解析）
			dataPoint := FlightDataPoint{
				Timestamp: time.Now(),
				Altitude:  5000.0 + float64(len(flightLog)*100), // 模拟数据
				Speed:     200.0 + float64(len(flightLog)*5),    // 模拟数据
				Mode:      1.0,
			}

			flightLog = append(flightLog, dataPoint)

			fmt.Printf("[%s] 记录点 #%d: 高度=%.0f, 速度=%.0f\n",
				dataPoint.Timestamp.Format("15:04:05"),
				len(flightLog),
				dataPoint.Altitude,
				dataPoint.Speed,
			)
		}
	}
}

/*
高级使用技巧:
==============

1. 错误处理最佳实践:
   - 始终检查连接状态
   - 实现重连机制
   - 记录详细的错误日志

2. 性能优化:
   - 使用批量操作减少 SimConnect 调用
   - 适当设置轮询间隔
   - 避免过度频繁的请求

3. 消息处理:
   - 实现专用的消息处理 goroutine
   - 使用通道进行消息传递
   - 实现超时机制

4. 生产环境建议:
   - 添加配置文件支持
   - 实现日志系统
   - 添加性能监控
   - 实现优雅关闭机制

5. 调试技巧:
   - 启用 SimConnect 日志
   - 使用 MobiFlight Connector 验证参数
   - 实现详细的调试输出
*/
