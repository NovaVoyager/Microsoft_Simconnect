package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 深入测试客户端数据区域
func main() {
	fmt.Println("=== 客户端数据区域深度测试 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("ClientData 测试")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")

	// 等待一下让初始消息处理完
	time.Sleep(500 * time.Millisecond)

	// 清空消息队列
	fmt.Println("清空消息队列...")
	for i := 0; i < 10; i++ {
		sc.GetNextDispatch()
	}
	fmt.Println()

	// 测试 1: 映射和创建数据区域
	fmt.Println("=== 测试 1: 映射客户端数据区域 ===")

	err = sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	fmt.Printf("MapClientDataNameToID(Command): ")
	if err != nil {
		fmt.Printf("失败 - %v\n", err)
	} else {
		fmt.Println("成功 ✓")
	}

	err = sc.MapClientDataNameToID("MobiFlight.Response", 0x00004243)
	fmt.Printf("MapClientDataNameToID(Response): ")
	if err != nil {
		fmt.Printf("失败 - %v\n", err)
	} else {
		fmt.Println("成功 ✓")
	}

	// 尝试创建命令区域
	fmt.Printf("\nCreateClientData(Command): ")
	err = sc.CreateClientData(0x00004242, 1024, 0)
	if err != nil {
		fmt.Printf("失败 - %v (可能已存在)\n", err)
	} else {
		fmt.Println("成功 ✓")
	}

	// 定义数据结构
	fmt.Printf("AddToClientDataDefinition(Command): ")
	err = sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)
	if err != nil {
		fmt.Printf("失败 - %v\n", err)
	} else {
		fmt.Println("成功 ✓")
	}

	fmt.Printf("AddToClientDataDefinition(Response): ")
	err = sc.AddToClientDataDefinition(0x10001, 0, 1024, 0, 0)
	if err != nil {
		fmt.Printf("失败 - %v\n", err)
	} else {
		fmt.Println("成功 ✓")
	}

	fmt.Println()

	// 测试 2: 请求客户端数据（不同的方式）
	fmt.Println("=== 测试 2: 请求客户端数据（多种方式） ===\n")

	testRequests := []struct {
		name   string
		period uint32
		flags  uint32
	}{
		{"ON_SET + DEFAULT", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 0},
		{"ON_SET + CHANGED", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 1},
		{"VISUAL_FRAME + DEFAULT", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_VISUAL_FRAME, 0},
		{"ONCE + DEFAULT", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE, 0},
	}

	for i, test := range testRequests {
		requestID := uint32(0x20000 + i)

		fmt.Printf("方式 %d: %s\n", i+1, test.name)
		fmt.Printf("  RequestID: 0x%08X\n", requestID)
		fmt.Printf("  Period: 0x%08X, Flags: 0x%08X\n", test.period, test.flags)

		err = sc.RequestClientData(
			0x00004243,  // Response 数据区域 ID
			requestID,   // 请求 ID
			0x10001,     // 定义 ID
			test.period, // Period
			test.flags,  // Flags
			0, 0, 0,
		)

		if err != nil {
			fmt.Printf("  ✗ RequestClientData 失败: %v\n\n", err)
			continue
		}

		fmt.Println("  ✓ RequestClientData 成功")

		// 立即检查是否有响应
		time.Sleep(200 * time.Millisecond)

		fmt.Println("  检查消息...")
		msgCount := 0
		for j := 0; j < 5; j++ {
			ppData, _, _ := sc.GetNextDispatch()
			if ppData != nil {
				msgCount++
				recv := (*simconnect.SIMCONNECT_RECV)(ppData)
				fmt.Printf("    收到消息类型: %d\n", recv.DwID)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
					fmt.Printf("    ✓✓ 客户端数据消息！RequestID=%d\n", clientData.RequestID)
				}
			}
		}
		if msgCount == 0 {
			fmt.Println("    没有收到消息")
		}
		fmt.Println()
	}

	// 测试 3: 发送命令并监听
	fmt.Println("=== 测试 3: 发送命令并监听响应 ===\n")

	// 先设置一个简单的请求
	requestID := uint32(0x30000)
	fmt.Printf("设置请求 (ID: 0x%08X)...\n", requestID)
	err = sc.RequestClientData(0x00004243, requestID, 0x10001,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 0, 0, 0, 0)

	if err != nil {
		fmt.Printf("✗ 失败: %v\n", err)
		return
	}
	fmt.Println("✓ 请求设置成功\n")

	time.Sleep(200 * time.Millisecond)

	// 发送命令
	fmt.Println("发送测试命令...")

	type Cmd struct {
		Code [1024]byte
	}

	testCommands := []string{
		"1 (>L:XMLVAR_Switch_AP_MSL_Mode)",
		"(L:XMLVAR_Switch_AP_MSL_Mode)",
		"MF.SimVars.Set.1 (>L:XMLVAR_Switch_AP_MSL_Mode)",
	}

	for i, cmdStr := range testCommands {
		fmt.Printf("\n命令 %d: %s\n", i+1, cmdStr)

		var cmd Cmd
		copy(cmd.Code[:], []byte(cmdStr))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ 发送失败: %v\n", err)
			continue
		}

		fmt.Println("  ✓ 发送成功")
		fmt.Println("  等待 2 秒并检查消息...")

		// 监听 2 秒
		foundResponse := false
		ticker := time.NewTicker(100 * time.Millisecond)
		timeout := time.After(2 * time.Second)

		msgCount := 0

	loop:
		for {
			select {
			case <-timeout:
				break loop
			case <-ticker.C:
				ppData, _, _ := sc.GetNextDispatch()
				if ppData == nil {
					continue
				}

				msgCount++
				recv := (*simconnect.SIMCONNECT_RECV)(ppData)

				fmt.Printf("  [消息 #%d] 类型: %d", msgCount, recv.DwID)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
					fmt.Printf(" (CLIENT_DATA, RequestID=0x%08X)\n", clientData.RequestID)

					if clientData.RequestID == requestID {
						foundResponse = true
						fmt.Println("  ✓✓✓ 这是我们的响应！")

						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)
						str := simconnect.ParseResponseString(response)
						fmt.Printf("  数据: '%s'\n", str)

						break loop
					}
				} else {
					fmt.Println()
				}
			}
		}
		ticker.Stop()

		if !foundResponse {
			if msgCount == 0 {
				fmt.Println("  ⚠ 完全没有收到消息")
			} else {
				fmt.Println("  ⚠ 收到消息但没有匹配的响应")
			}
		}
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("\n诊断建议:")
	fmt.Println("1. 如果 RequestClientData 失败:")
	fmt.Println("   → MobiFlight WASM 模块未创建 Response 数据区域")
	fmt.Println("   → 尝试使用 MobiFlight Connector 启动一次连接")
	fmt.Println()
	fmt.Println("2. 如果完全没有收到消息:")
	fmt.Println("   → 飞机可能不支持这些 LVar")
	fmt.Println("   → 尝试在游戏中改变一些设置（如自动驾驶）")
	fmt.Println()
	fmt.Println("3. 如果收到其他消息但没有 CLIENT_DATA:")
	fmt.Println("   → WASM 模块可能没有响应我们的命令")
	fmt.Println("   → 命令格式可能不正确")
	fmt.Println()
}
