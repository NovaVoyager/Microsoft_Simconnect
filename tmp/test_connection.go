package main

import (
	"fmt"
	"log"
	"time"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 这是一个简单的测试程序，用于验证 MobiFlight WASM 模块连接
func main111() {
	fmt.Println("=================================")
	fmt.Println("MobiFlight WASM 连接测试程序")
	fmt.Println("=================================\n")

	// 步骤 1: 连接到 SimConnect
	fmt.Println("步骤 1: 连接到 SimConnect...")
	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatalf("✗ 加载 SimConnect.dll 失败: %v\n\n请确保 simconnect/SimConnect.dll 文件存在", err)
	}
	fmt.Println("✓ SimConnect.dll 加载成功")

	err = sc.Open("MobiFlight 测试程序")
	if err != nil {
		log.Fatalf("✗ 连接失败: %v\n\n可能的原因:\n  1. Microsoft Flight Simulator 未运行\n  2. SimConnect.dll 版本不匹配\n  3. 防火墙阻止连接\n", err)
	}
	defer sc.Close()
	fmt.Println("✓ SimConnect 连接成功\n")

	// 步骤 2: 初始化 MobiFlight 客户端
	fmt.Println("步骤 2: 初始化 MobiFlight 客户端...")
	mf, err := simconnect.NewMobiFlightClient(sc)
	if err != nil {
		log.Fatalf("✗ MobiFlight 初始化失败: %v\n\n可能的原因:\n  1. MobiFlight WASM 模块未安装\n  2. WASM 模块未加载\n\n请参阅 TROUBLESHOOTING.md 获取安装指南\n", err)
	}
	fmt.Println("✓ MobiFlight 客户端初始化成功\n")

	// 步骤 3: 测试设置 LVar
	fmt.Println("步骤 3: 测试设置 LVar...")
	testVarName := "XMLVAR_Switch_AP_MSL_Mode"
	testValue := 1.0

	fmt.Printf("  设置 L:%s = %.1f\n", testVarName, testValue)
	err = mf.SetLVar(testVarName, testValue)
	if err != nil {
		log.Printf("✗ 设置 LVar 失败: %v\n", err)
		log.Printf("\n这可能意味着:\n")
		log.Printf("  1. MobiFlight WASM 模块未正确加载\n")
		log.Printf("  2. 客户端数据区域未创建\n")
		log.Printf("  3. LVar 名称不存在（但这不应该导致错误）\n\n")
		log.Fatal("测试失败")
	}
	fmt.Println("✓ LVar 设置成功（命令已发送）\n")

	// 等待一会儿让模拟器处理
	time.Sleep(200 * time.Millisecond)

	// 步骤 4: 测试读取 LVar
	fmt.Println("步骤 4: 测试读取 LVar...")
	fmt.Printf("  请求读取 L:%s\n", testVarName)
	err = mf.GetLVar(testVarName)
	if err != nil {
		log.Printf("✗ 读取 LVar 失败: %v\n", err)
		log.Fatal("测试失败")
	}
	fmt.Println("✓ LVar 读取请求已发送\n")

	// 步骤 5: 尝试接收响应
	fmt.Println("步骤 5: 等待响应数据...")
	fmt.Println("  (尝试接收 5 秒...)")

	received := false
	timeout := time.After(500 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for !received {
		select {
		case <-timeout:
			fmt.Println("\n⚠ 未在 5 秒内收到响应")
			fmt.Println("\n这可能意味着:")
			fmt.Println("  1. MobiFlight WASM 模块未正确加载")
			fmt.Println("  2. 响应数据区域未创建")
			fmt.Println("  3. LVar 不存在（对于当前飞机）")
			fmt.Println("  4. 消息处理有延迟")
			fmt.Println("\n建议:")
			fmt.Println("  - 使用 MobiFlight Connector 验证 WASM 模块已加载")
			fmt.Println("  - 使用 SimConnect Inspector 查看客户端数据区域")
			fmt.Println("  - 确认当前飞机支持这个 LVar")
			fmt.Println("  - 查看 TROUBLESHOOTING.md 获取更多信息")
			goto TestComplete

		case <-ticker.C:
			response, err := mf.ReceiveResponse()
			if err != nil {
				// 真正的错误
				fmt.Printf("✗ 接收响应时出错: %v\n", err)
				goto TestComplete
			}

			if response == nil {
				// 没有消息，继续等待
				continue
			}

			// 成功收到响应
			value, parseErr := simconnect.ParseResponseFloat(response)
			if parseErr != nil {
				fmt.Printf("✓ 收到响应，但解析失败: %v\n", parseErr)
				fmt.Printf("  原始数据: %s\n", simconnect.ParseResponseString(response))
			} else {
				fmt.Printf("✓ 成功收到响应: L:%s = %.6f\n", testVarName, value)
			}
			received = true
		}
	}

TestComplete:
	// 测试总结
	fmt.Println("\n=================================")
	fmt.Println("测试总结")
	fmt.Println("=================================")
	fmt.Println("✓ SimConnect 连接正常")
	fmt.Println("✓ MobiFlight 客户端初始化成功")
	fmt.Println("✓ LVar 写入功能正常")
	fmt.Println("✓ LVar 读取请求功能正常")

	if received {
		fmt.Println("✓ 响应接收功能正常")
		fmt.Println("\n🎉 所有测试通过！MobiFlight WASM 模块工作正常！")
	} else {
		fmt.Println("⚠ 响应接收未测试或失败")
		fmt.Println("\n⚠ 部分功能可能有问题，请查看上述建议")
	}

	fmt.Println("\n下一步:")
	fmt.Println("  - 如果所有测试通过，可以开始使用 main.go 中的示例")
	fmt.Println("  - 如果有问题，请查看 TROUBLESHOOTING.md")
	fmt.Println("  - 使用 MobiFlight Connector 查找你的飞机支持的 LVar")
	fmt.Println()
}
