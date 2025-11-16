package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试客户端数据区域的基本功能（不依赖 MobiFlight）
func main() {
	fmt.Println("=== 客户端数据区域基础功能测试 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("基础测试")
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

	// 测试 1: 创建我们自己的测试数据区域
	fmt.Println("=== 测试 1: 创建自己的客户端数据区域 ===\n")

	testDataAreaID := uint32(0x12345678)
	testDefineID := uint32(0x1000)
	testRequestID := uint32(0x2000)

	// 1.1 映射名称
	fmt.Println("1. MapClientDataNameToID...")
	err = sc.MapClientDataNameToID("MyTestArea", testDataAreaID)
	if err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
		return
	}
	fmt.Println("   ✓ 成功\n")

	// 1.2 创建数据区域
	fmt.Println("2. CreateClientData...")
	err = sc.CreateClientData(testDataAreaID, 256, 0)
	if err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
		return
	}
	fmt.Println("   ✓ 成功\n")

	// 1.3 定义数据结构
	fmt.Println("3. AddToClientDataDefinition...")
	err = sc.AddToClientDataDefinition(testDefineID, 0, 256, 0, 0)
	if err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
		return
	}
	fmt.Println("   ✓ 成功\n")

	// 1.4 请求数据
	fmt.Println("4. RequestClientData...")
	err = sc.RequestClientData(testDataAreaID, testRequestID, testDefineID,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 0, 0, 0, 0)
	if err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
		return
	}
	fmt.Println("   ✓ 成功\n")

	// 1.5 写入数据
	fmt.Println("5. SetClientData (写入测试数据)...")

	type TestData struct {
		Data [256]byte
	}

	var testData TestData
	testString := "Hello from Go!"
	copy(testData.Data[:], []byte(testString))

	err = sc.SetClientData(testDataAreaID, testDefineID, 0, 0, 256, unsafe.Pointer(&testData))
	if err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
		return
	}
	fmt.Println("   ✓ 成功\n")

	// 1.6 尝试接收数据（我们自己写的）
	fmt.Println("6. 尝试接收客户端数据消息...")
	time.Sleep(200 * time.Millisecond)

	foundOwnData := false
	for i := 0; i < 10; i++ {
		ppData, _, _ := sc.GetNextDispatch()
		if ppData == nil {
			continue
		}

		recv := (*simconnect.SIMCONNECT_RECV)(ppData)
		fmt.Printf("   收到消息类型: %d", recv.DwID)

		if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
			clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
			fmt.Printf(" (CLIENT_DATA, RequestID=0x%08X)\n", clientData.RequestID)

			if clientData.RequestID == testRequestID {
				foundOwnData = true
				fmt.Println("   ✓✓ 收到了我们自己写入的数据！")

				dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
				receivedData := (*TestData)(dataPtr)

				// 查找字符串结束
				endIdx := 0
				for j, b := range receivedData.Data {
					if b == 0 {
						endIdx = j
						break
					}
				}

				receivedString := string(receivedData.Data[:endIdx])
				fmt.Printf("   接收到的数据: '%s'\n", receivedString)

				if receivedString == testString {
					fmt.Println("   ✓✓✓ 数据匹配！客户端数据区域功能正常！\n")
				}
				break
			}
		} else {
			fmt.Println()
		}
	}

	if !foundOwnData {
		fmt.Println("   ⚠ 未收到客户端数据消息")
		fmt.Println("   这很奇怪 - 我们自己创建的数据区域也收不到数据\n")
	}

	// 测试 2: 尝试访问 MobiFlight 的数据区域
	fmt.Println("=== 测试 2: 尝试访问 MobiFlight 数据区域 ===\n")

	fmt.Println("尝试 RequestClientData 到 MobiFlight.Response...")

	mfResponseID := uint32(0x00004243)
	mfDefineID := uint32(0x10001)
	mfRequestID := uint32(0x30000)

	// 先定义
	sc.AddToClientDataDefinition(mfDefineID, 0, 1024, 0, 0)

	// 尝试请求
	err = sc.RequestClientData(mfResponseID, mfRequestID, mfDefineID,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET, 0, 0, 0, 0)

	if err != nil {
		fmt.Printf("✗ RequestClientData 失败: %v\n", err)
		fmt.Println("\n这证实了 MobiFlight.Response 数据区域不存在或未被创建。\n")
	} else {
		fmt.Println("✓ RequestClientData 成功")
		fmt.Println("MobiFlight.Response 数据区域存在！\n")

		// 尝试向 MobiFlight.Command 发送数据
		fmt.Println("向 MobiFlight.Command 发送测试命令...")

		sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
		sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)

		type MFCmd struct {
			Code [1024]byte
		}

		testCommands := []string{
			"(L:XMLVAR_Switch_AP_MSL_Mode)",
			"1 (>L:XMLVAR_Switch_AP_MSL_Mode)",
		}

		for _, cmdStr := range testCommands {
			fmt.Printf("\n命令: %s\n", cmdStr)

			var cmd MFCmd
			copy(cmd.Code[:], []byte(cmdStr))

			err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
			if err != nil {
				fmt.Printf("  ✗ 发送失败: %v\n", err)
			} else {
				fmt.Println("  ✓ 发送成功，等待响应...")

				time.Sleep(500 * time.Millisecond)

				for i := 0; i < 10; i++ {
					ppData, _, _ := sc.GetNextDispatch()
					if ppData == nil {
						continue
					}

					recv := (*simconnect.SIMCONNECT_RECV)(ppData)
					if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
						clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

						if clientData.RequestID == mfRequestID {
							fmt.Println("  ✓✓ 收到 MobiFlight 响应！")

							dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
							response := (*simconnect.MobiFlightResponse)(dataPtr)
							str := simconnect.ParseResponseString(response)
							fmt.Printf("  数据: '%s'\n", str)
							break
						}
					}
				}
			}
		}
	}

	fmt.Println("\n=== 测试完成 ===\n")

	fmt.Println("诊断结果:")
	if foundOwnData {
		fmt.Println("✓ 客户端数据区域基本功能正常")
		fmt.Println("  → SimConnect 本身工作正常")
		fmt.Println("  → 问题在于 MobiFlight WASM 模块的配置\n")

		fmt.Println("可能的原因:")
		fmt.Println("  1. MobiFlight WASM 模块未创建 Response 数据区域")
		fmt.Println("  2. 需要发送特定的注册/激活命令")
		fmt.Println("  3. WASM 模块版本不兼容")
		fmt.Println("  4. 需要在特定的游戏状态下（如飞行中）")
	} else {
		fmt.Println("⚠ 客户端数据区域基本功能异常")
		fmt.Println("  → 即使是自己创建的数据区域也收不到数据")
		fmt.Println("  → 可能的 SimConnect 配置问题\n")

		fmt.Println("建议:")
		fmt.Println("  1. 确保进入了飞行（不只是主菜单）")
		fmt.Println("  2. 检查 SimConnect.cfg 配置")
		fmt.Println("  3. 尝试重启 MSFS")
	}

	fmt.Println()
}
