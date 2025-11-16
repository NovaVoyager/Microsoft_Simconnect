package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 这个程序验证 MobiFlight WASM 模块的配置
func main() {
	fmt.Println("=================================")
	fmt.Println("MobiFlight WASM 配置验证工具")
	fmt.Println("=================================\n")

	// 连接到 SimConnect
	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatalf("加载 DLL 失败: %v", err)
	}

	err = sc.Open("MobiFlight 验证工具")
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")

	// 显示我们使用的配置
	fmt.Println("=== 当前配置 ===")
	fmt.Printf("命令数据区域名称: %s\n", simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_COMMAND)
	fmt.Printf("响应数据区域名称: %s\n", simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_RESPONSE)
	fmt.Printf("命令数据区域 ID: 0x%08X (%d)\n", simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND)
	fmt.Printf("响应数据区域 ID: 0x%08X (%d)\n", simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE)
	fmt.Printf("命令定义 ID: 0x%08X (%d)\n", simconnect.MOBIFLIGHT_DEFINE_ID_COMMAND, simconnect.MOBIFLIGHT_DEFINE_ID_COMMAND)
	fmt.Printf("响应定义 ID: 0x%08X (%d)\n", simconnect.MOBIFLIGHT_DEFINE_ID_RESPONSE, simconnect.MOBIFLIGHT_DEFINE_ID_RESPONSE)
	fmt.Printf("响应请求 ID: 0x%08X (%d)\n", simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE, simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE)
	fmt.Printf("消息大小: %d 字节\n\n", simconnect.MOBIFLIGHT_MESSAGE_SIZE)

	// 步骤 1: 映射客户端数据区域名称
	fmt.Println("=== 步骤 1: 映射客户端数据区域 ===")

	err = sc.MapClientDataNameToID(simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_COMMAND, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND)
	if err != nil {
		fmt.Printf("✗ 映射命令区域失败: %v\n", err)
	} else {
		fmt.Printf("✓ 映射命令区域成功: %s -> 0x%08X\n", simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_COMMAND, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND)
	}

	err = sc.MapClientDataNameToID(simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_RESPONSE, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE)
	if err != nil {
		fmt.Printf("✗ 映射响应区域失败: %v\n", err)
	} else {
		fmt.Printf("✓ 映射响应区域成功: %s -> 0x%08X\n", simconnect.MOBIFLIGHT_CLIENT_DATA_NAME_RESPONSE, simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE)
	}

	fmt.Println()

	// 步骤 2: 尝试创建客户端数据区域
	fmt.Println("=== 步骤 2: 创建客户端数据区域 ===")

	err = sc.CreateClientData(simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND, simconnect.MOBIFLIGHT_MESSAGE_SIZE, 0)
	if err != nil {
		fmt.Printf("⚠ 创建命令区域失败: %v (可能已存在，这是正常的)\n", err)
	} else {
		fmt.Printf("✓ 创建命令区域成功\n")
	}

	// 注意：响应区域应该由 WASM 模块创建，所以我们不创建它
	fmt.Println("  (跳过创建响应区域 - 应由 WASM 模块创建)")
	fmt.Println()

	// 步骤 3: 定义数据结构
	fmt.Println("=== 步骤 3: 定义数据结构 ===")

	err = sc.AddToClientDataDefinition(simconnect.MOBIFLIGHT_DEFINE_ID_COMMAND, 0, simconnect.MOBIFLIGHT_MESSAGE_SIZE, 0, 0)
	if err != nil {
		fmt.Printf("✗ 定义命令数据结构失败: %v\n", err)
	} else {
		fmt.Printf("✓ 定义命令数据结构成功\n")
	}

	err = sc.AddToClientDataDefinition(simconnect.MOBIFLIGHT_DEFINE_ID_RESPONSE, 0, simconnect.MOBIFLIGHT_MESSAGE_SIZE, 0, 0)
	if err != nil {
		fmt.Printf("✗ 定义响应数据结构失败: %v\n", err)
	} else {
		fmt.Printf("✓ 定义响应数据结构成功\n")
	}

	fmt.Println()

	// 步骤 4: 请求接收响应数据
	fmt.Println("=== 步骤 4: 请求接收响应数据 ===")

	err = sc.RequestClientData(
		simconnect.MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE,
		simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE,
		simconnect.MOBIFLIGHT_DEFINE_ID_RESPONSE,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
		simconnect.SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED,
		0, 0, 0,
	)
	if err != nil {
		fmt.Printf("✗ 请求响应数据失败: %v\n", err)
		fmt.Println("\n⚠ 这可能意味着 MobiFlight WASM 模块未创建响应数据区域！")
	} else {
		fmt.Printf("✓ 请求响应数据成功\n")
	}

	fmt.Println()

	// 步骤 5: 发送一个测试命令
	fmt.Println("=== 步骤 5: 发送测试命令 ===")

	// 准备命令数据
	type MobiFlightCommand struct {
		Code [1024]byte
	}

	var cmd MobiFlightCommand

	// 测试不同的命令格式
	testCommands := []string{
		"(L:XMLVAR_Switch_AP_MSL_Mode)",    // RPN 格式 - 读取
		"1 (>L:XMLVAR_Switch_AP_MSL_Mode)", // RPN 格式 - 设置
		"MF.SimVars.Set.1 (>L:TEST_VAR)",   // MobiFlight 格式 1
		"MF.DummyCmd",                      // 测试命令
	}

	for i, cmdStr := range testCommands {
		fmt.Printf("\n测试命令 %d: %s\n", i+1, cmdStr)

		// 清空命令缓冲区
		for j := range cmd.Code {
			cmd.Code[j] = 0
		}

		// 复制命令字符串
		cmdBytes := []byte(cmdStr)
		copy(cmd.Code[:], cmdBytes)

		// 发送命令
		err = sc.SetClientData(
			simconnect.MOBIFLIGHT_CLIENT_DATA_ID_COMMAND,
			simconnect.MOBIFLIGHT_DEFINE_ID_COMMAND,
			simconnect.SIMCONNECT_CLIENT_DATA_SET_FLAG_DEFAULT,
			0,
			simconnect.MOBIFLIGHT_MESSAGE_SIZE,
			unsafe.Pointer(&cmd),
		)

		if err != nil {
			fmt.Printf("  ✗ 发送失败: %v\n", err)
		} else {
			fmt.Printf("  ✓ 发送成功\n")

			// 等待一会儿
			time.Sleep(200 * time.Millisecond)

			// 尝试接收响应
			fmt.Printf("  等待响应...\n")

			foundResponse := false
			for j := 0; j < 10; j++ {
				ppData, _, err := sc.GetNextDispatch()
				if err != nil {
					fmt.Printf("  ✗ 接收错误: %v\n", err)
					break
				}

				if ppData == nil {
					time.Sleep(50 * time.Millisecond)
					continue
				}

				recv := (*simconnect.SIMCONNECT_RECV)(ppData)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

					if clientData.RequestID == simconnect.MOBIFLIGHT_REQUEST_ID_RESPONSE {
						foundResponse = true
						fmt.Printf("  ✓✓ 收到响应！\n")

						// 读取数据
						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)

						str := simconnect.ParseResponseString(response)
						fmt.Printf("  响应数据: '%s'\n", str)

						if str != "" {
							value, err := simconnect.ParseResponseFloat(response)
							if err == nil {
								fmt.Printf("  解析为数字: %.6f\n", value)
							}
						}
						break
					}
				}
			}

			if !foundResponse {
				fmt.Printf("  ⚠ 未收到响应\n")
			}
		}
	}

	fmt.Println("\n=================================")
	fmt.Println("验证完成")
	fmt.Println("=================================\n")

	fmt.Println("建议:")
	fmt.Println("1. 如果所有步骤都成功，但仍未收到响应：")
	fmt.Println("   - 尝试在游戏中执行一些操作（例如改变自动驾驶设置）")
	fmt.Println("   - 使用 MobiFlight Connector 测试相同的 LVar")
	fmt.Println("   - 检查 LVar 名称是否正确")
	fmt.Println()
	fmt.Println("2. 如果 RequestClientData 失败：")
	fmt.Println("   - MobiFlight WASM 模块可能未创建响应数据区域")
	fmt.Println("   - 尝试重启 MSFS")
	fmt.Println("   - 重新安装 MobiFlight WASM 模块")
	fmt.Println()
	fmt.Println("3. 如果发送命令失败：")
	fmt.Println("   - 命令数据区域可能未正确创建")
	fmt.Println("   - 检查 SimConnect 日志")
	fmt.Println()
}
