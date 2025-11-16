package main

import (
	"fmt"
	"log"
	"time"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

func main() {
	fmt.Println("=== MobiFlight WASM 模块 Go 客户端示例 ===\n")

	// 运行示例
	if err := exampleMessageLoop(); err != nil {
		log.Fatalf("示例失败: %v", err)
	}
}

// exampleBasicUsage 演示基本的 LVar 读取和设置
func exampleBasicUsage() error {
	fmt.Println("示例 1: 基本的 LVar 读取和设置")
	fmt.Println("--------------------------------")

	// 1. 创建并初始化 SimConnect 和 MobiFlight 客户端
	sc, mf, err := simconnect.SetupMobiFlightClient("MobiFlight Go Client")
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功")
	fmt.Println("✓ MobiFlight 客户端初始化完成\n")

	// 2. 设置一个 LVar 的值
	fmt.Println("步骤 1: 设置 LVar 的值")
	varName := "XMLVAR_Switch_AP_MSL_Mode"
	newValue := 1.0

	fmt.Printf("  设置 L:%s = %.1f\n", varName, newValue)
	if err := mf.SetLVar(varName, newValue); err != nil {
		return fmt.Errorf("设置 LVar 失败: %v", err)
	}
	fmt.Println("  ✓ LVar 设置成功\n")

	// 等待一小段时间让模拟器处理
	time.Sleep(100 * time.Millisecond)

	// 3. 读取 LVar 的值
	fmt.Println("步骤 2: 读取 LVar 的值")
	fmt.Printf("  请求读取 L:%s\n", varName)
	if err := mf.GetLVar(varName); err != nil {
		return fmt.Errorf("请求 LVar 失败: %v", err)
	}

	// 注意: 在实际应用中，你需要在消息循环中调用 ReceiveResponse()
	// 来接收响应。这里为了演示简化了流程。
	fmt.Println("  ✓ LVar 读取请求已发送")
	fmt.Println("  (需要在消息循环中调用 ReceiveResponse() 来获取实际值)\n")

	fmt.Println("✓ 示例 1 完成\n")
	return nil
}

// exampleBatchOperations 演示批量操作 LVar
func exampleBatchOperations() error {
	fmt.Println("示例 2: 批量设置多个 LVar")
	fmt.Println("--------------------------------")

	// 创建客户端
	sc, mf, err := simconnect.SetupMobiFlightClient("MobiFlight Go Batch Client")
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ 客户端初始化完成\n")

	// 创建批量操作对象
	batch := simconnect.NewLVarBatch(mf)

	// 添加多个 LVar 设置操作
	batch.Add("XMLVAR_Switch_AP_MSL_Mode", 1.0)
	batch.Add("XMLVAR_Autopilot_Altitude", 5000.0)
	batch.Add("XMLVAR_Airspeed_Mode", 2.0)

	fmt.Println("批量设置以下 LVar:")
	fmt.Println("  - L:XMLVAR_Switch_AP_MSL_Mode = 1.0")
	fmt.Println("  - L:XMLVAR_Autopilot_Altitude = 5000.0")
	fmt.Println("  - L:XMLVAR_Airspeed_Mode = 2.0\n")

	// 执行批量操作
	if err := batch.Execute(); err != nil {
		return fmt.Errorf("批量操作失败: %v", err)
	}

	fmt.Println("✓ 批量操作完成\n")
	return nil
}

// exampleRPNCode 演示执行自定义 RPN 计算器代码
func exampleRPNCode() error {
	fmt.Println("示例 3: 执行自定义 RPN 代码")
	fmt.Println("--------------------------------")

	// 创建客户端
	sc, mf, err := simconnect.SetupMobiFlightClient("MobiFlight Go RPN Client")
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ 客户端初始化完成\n")

	// 执行 RPN 代码
	// 示例: 将两个值相加并存储到 LVar
	rpnCode := "100 200 + (>L:MY_CUSTOM_VAR)"
	fmt.Printf("执行 RPN 代码: %s\n", rpnCode)
	fmt.Println("(这将计算 100 + 200 并将结果 300 存储到 L:MY_CUSTOM_VAR)\n")

	if err := mf.ExecuteCode(rpnCode); err != nil {
		return fmt.Errorf("执行 RPN 代码失败: %v", err)
	}

	fmt.Println("✓ RPN 代码执行成功\n")
	return nil
}

// exampleMessageLoop 演示完整的消息循环（接收响应）
func exampleMessageLoop() error {
	fmt.Println("示例 4: 完整的消息循环示例")
	fmt.Println("--------------------------------")

	// 创建客户端
	sc, mf, err := simconnect.SetupMobiFlightClient("MobiFlight Go Loop Client")
	if err != nil {
		return fmt.Errorf("初始化客户端失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ 客户端初始化完成\n")

	// 请求读取一个 LVar
	varName := "INI_APU_RUN"
	fmt.Printf("请求读取 L:%s\n", varName)
	if err := mf.GetLVar(varName); err != nil {
		return fmt.Errorf("请求 LVar 失败: %v", err)
	}

	// 消息循环
	fmt.Println("\n开始消息循环（等待响应）...")
	fmt.Println("(按 Ctrl+C 退出)\n")

	timeout := time.After(500000 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("超时，未收到响应")
			return nil

		case <-ticker.C:
			// 尝试接收响应
			response, err := mf.ReceiveResponse()
			if err != nil {
				// 真正的错误
				fmt.Printf("接收响应时出错: %v\n", err)
				return err
			}

			if response == nil {
				// 没有消息，继续等待
				continue
			}

			// 解析响应
			value, err := simconnect.ParseResponseFloat(response)
			if err != nil {
				fmt.Printf("解析响应失败: %v\n", err)
				continue
			}

			fmt.Printf("✓ 收到响应: L:%s = %.6f\n", varName, value)
			return nil
		}
	}
}

/*
使用说明:
==========

1. 前提条件:
   - Microsoft Flight Simulator 正在运行
   - MobiFlight WASM 模块已安装在模拟器中
   - simconnect/SimConnect.dll 文件存在

2. 编译和运行:
   go build -o mobiflight_example.exe main.go
   ./mobiflight_example.exe

3. 常见的 LVar 示例:
   - L:XMLVAR_Switch_AP_MSL_Mode - 自动驾驶模式开关
   - L:XMLVAR_Autopilot_Altitude - 自动驾驶高度设置
   - L:XMLVAR_Airspeed_Mode - 空速模式
   - L:XMLVAR_Switch_EFIS_Range - EFIS 范围设置

4. RPN 计算器代码说明:
   RPN (Reverse Polish Notation) 是一种后缀表达式，例如:
   - "100 200 +" 表示 100 + 200
   - "1 (>L:MY_VAR)" 表示将 1 写入 L:MY_VAR
   - "(L:MY_VAR) 10 +" 表示读取 L:MY_VAR 并加 10

5. 工作原理:
   - MobiFlight WASM 模块在模拟器中运行，提供访问 LVar 的接口
   - 通过 SimConnect 的 Client Data Area (CDA) 机制通信
   - 命令通过 MOBIFLIGHT_CLIENT_DATA_ID_COMMAND 发送
   - 响应通过 MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE 接收

6. 注意事项:
   - SetLVar() 立即发送命令，无需等待响应
   - GetLVar() 需要在消息循环中调用 ReceiveResponse() 来获取结果
   - 批量操作可以提高性能，减少 SimConnect 调用次数
   - 确保 LVar 名称正确，否则模拟器不会响应

7. 调试技巧:
   - 使用 MobiFlight Connector 查看可用的 LVar 列表
   - 检查 SimConnect.log 文件排查连接问题
   - 确认 WASM 模块已正确安装并加载
*/
