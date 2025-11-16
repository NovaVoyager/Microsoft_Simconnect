package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 检查 WASM 模块的状态
func main() {
	fmt.Println("=== WASM 模块状态检查 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("WASM 检查")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")
	time.Sleep(300 * time.Millisecond)

	// 清空初始消息
	for i := 0; i < 10; i++ {
		sc.GetNextDispatch()
	}

	fmt.Println("=== 检查 MobiFlight 数据区域 ===\n")

	// 尝试各种可能的数据区域名称和 ID
	testCases := []struct {
		name        string
		areaName    string
		areaID      uint32
		shouldExist bool
	}{
		{"MobiFlight.Command", "MobiFlight.Command", 0x00004242, true},
		{"MobiFlight.Response", "MobiFlight.Response", 0x00004243, false}, // 由 WASM 创建
		{"MobiFlight.LVars", "MobiFlight.LVars", 0x00004244, false},       // 由 WASM 创建
		{"MobiFlight.SimVars", "MobiFlight.SimVars", 0x00004245, false},
	}

	for i, tc := range testCases {
		fmt.Printf("%d. 检查 %s (ID: 0x%08X)...\n", i+1, tc.name, tc.areaID)

		// 映射名称
		err = sc.MapClientDataNameToID(tc.areaName, tc.areaID)
		if err != nil {
			fmt.Printf("   MapClientDataNameToID 失败: %v\n", err)
			continue
		}

		// 定义数据结构
		defineID := uint32(0x20000 + i)
		err = sc.AddToClientDataDefinition(defineID, 0, 1024, 0, 0)
		if err != nil {
			fmt.Printf("   AddToClientDataDefinition 失败: %v\n", err)
			continue
		}

		// 尝试请求数据
		requestID := uint32(0x30000 + i)
		err = sc.RequestClientData(tc.areaID, requestID, defineID,
			simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE, 0, 0, 0, 0)

		if err != nil {
			fmt.Printf("   ✗ RequestClientData 失败: %v\n", err)
			fmt.Println("   → 数据区域不存在或未被创建")
		} else {
			fmt.Println("   ✓ RequestClientData 成功")
			fmt.Println("   → 数据区域存在！")

			// 如果这是 Command 区域，尝试写入
			if tc.name == "MobiFlight.Command" {
				fmt.Println("   尝试写入测试命令...")

				type Cmd struct {
					Code [1024]byte
				}
				var cmd Cmd
				copy(cmd.Code[:], []byte("MF.Ping"))

				err = sc.SetClientData(tc.areaID, defineID, 0, 0, 1024, unsafe.Pointer(&cmd))
				if err != nil {
					fmt.Printf("   ✗ SetClientData 失败: %v\n", err)
				} else {
					fmt.Println("   ✓ SetClientData 成功")
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("=== 尝试不同的激活方法 ===\n")

	// 首先映射 Command 区域
	sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	sc.CreateClientData(0x00004242, 1024, 0)
	sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)

	activationCommands := []string{
		"MF.Clients.Add.GoClient",
		"MF.SimVars.Clear",
		"MF.DummyCallback",
		"",
	}

	for i, cmd := range activationCommands {
		if cmd == "" {
			fmt.Printf("%d. 发送空命令\n", i+1)
		} else {
			fmt.Printf("%d. 发送命令: %s\n", i+1, cmd)
		}

		type CmdData struct {
			Code [1024]byte
		}
		var cmdData CmdData
		copy(cmdData.Code[:], []byte(cmd))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmdData))
		if err != nil {
			fmt.Printf("   ✗ 发送失败: %v\n", err)
		} else {
			fmt.Println("   ✓ 发送成功")

			// 等待并检查是否创建了 Response 区域
			time.Sleep(500 * time.Millisecond)

			testDefineID := uint32(0x50000 + i)
			testRequestID := uint32(0x60000 + i)

			sc.AddToClientDataDefinition(testDefineID, 0, 1024, 0, 0)
			err = sc.RequestClientData(0x00004243, testRequestID, testDefineID,
				simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE, 0, 0, 0, 0)

			if err == nil {
				fmt.Println("   ✓✓ Response 区域现在可用了！")
				fmt.Printf("   → 激活命令可能是: '%s'\n", cmd)

				// 检查是否有响应
				time.Sleep(200 * time.Millisecond)
				for j := 0; j < 5; j++ {
					ppData, _, _ := sc.GetNextDispatch()
					if ppData != nil {
						recv := (*simconnect.SIMCONNECT_RECV)(ppData)
						if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
							fmt.Println("   ✓✓✓ 收到客户端数据消息！")
						}
					}
				}
			}
		}

		fmt.Println()
	}

	fmt.Println("=== 检查完成 ===\n")

	fmt.Println("总结:")
	fmt.Println("如果所有 RequestClientData 都失败:")
	fmt.Println("  → MobiFlight WASM 模块可能:")
	fmt.Println("    - 未运行")
	fmt.Println("    - 使用不同的通信机制")
	fmt.Println("    - 需要特定版本的 MSFS")
	fmt.Println()
	fmt.Println("如果 Command 区域存在但 Response 不存在:")
	fmt.Println("  → WASM 模块未激活响应机制")
	fmt.Println("  → 需要找到正确的激活命令")
	fmt.Println()
}
