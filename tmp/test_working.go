package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 完整的工作示例
func main() {
	fmt.Println("=== MobiFlight 完整工作示例 ===\n")

	// 1. 连接
	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("MobiFlight Go Client")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	fmt.Println("✓ SimConnect 连接成功\n")

	// 等待初始化
	time.Sleep(500 * time.Millisecond)

	// 清空初始消息
	for i := 0; i < 20; i++ {
		sc.GetNextDispatch()
	}

	// 2. 映射数据区域
	fmt.Println("初始化 MobiFlight 数据区域...")

	sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	sc.MapClientDataNameToID("MobiFlight.Response", 0x00004243)

	// 注意：不需要 CreateClientData，因为 WASM 模块已经创建了

	// 3. 定义数据结构
	sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0) // Command
	sc.AddToClientDataDefinition(0x10001, 0, 1024, 0, 0) // Response

	// 4. 请求接收响应数据
	err = sc.RequestClientData(
		0x00004243, // Response 区域
		0x20000,    // Request ID
		0x10001,    // Define ID
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
		simconnect.SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED,
		0, 0, 0,
	)

	if err != nil {
		log.Fatalf("RequestClientData 失败: %v", err)
	}

	fmt.Println("✓ MobiFlight 初始化完成\n")

	// 等待一下
	time.Sleep(300 * time.Millisecond)

	// 5. 测试：设置一个 LVar
	fmt.Println("=== 测试 1: 设置 LVar ===")

	testVar := "XMLVAR_Switch_AP_MSL_Mode"
	testValue := 1.0

	fmt.Printf("设置 L:%s = %.1f\n", testVar, testValue)

	type MFCommand struct {
		Code [1024]byte
	}

	var cmd MFCommand
	cmdStr := fmt.Sprintf("%.6f (>L:%s)", testValue, testVar)
	copy(cmd.Code[:], []byte(cmdStr))

	err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
	if err != nil {
		log.Fatalf("SetClientData 失败: %v", err)
	}

	fmt.Println("✓ 命令已发送\n")

	time.Sleep(500 * time.Millisecond)

	// 6. 检查是否有响应
	fmt.Println("检查响应消息...")
	for i := 0; i < 10; i++ {
		ppData, _, _ := sc.GetNextDispatch()
		if ppData != nil {
			recv := (*simconnect.SIMCONNECT_RECV)(ppData)
			fmt.Printf("收到消息类型: %d\n", recv.DwID)

			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				fmt.Println("✓ 这是客户端数据消息")
			}
		}
	}

	fmt.Println()

	// 7. 测试：读取 LVar
	fmt.Println("=== 测试 2: 读取 LVar ===")

	fmt.Printf("读取 L:%s\n", testVar)

	var cmd2 MFCommand
	cmdStr2 := fmt.Sprintf("(L:%s)", testVar)
	copy(cmd2.Code[:], []byte(cmdStr2))

	err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd2))
	if err != nil {
		log.Fatalf("SetClientData 失败: %v", err)
	}

	fmt.Println("✓ 读取命令已发送")
	fmt.Println("等待响应...\n")

	// 持续监听响应
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	foundResponse := false

	for !foundResponse {
		select {
		case <-timeout:
			fmt.Println("⚠ 5 秒内未收到响应")
			goto TestComplete

		case <-ticker.C:
			ppData, _, _ := sc.GetNextDispatch()
			if ppData == nil {
				continue
			}

			recv := (*simconnect.SIMCONNECT_RECV)(ppData)

			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

				fmt.Printf("收到客户端数据消息 (RequestID: 0x%08X)\n", clientData.RequestID)

				if clientData.RequestID == 0x20000 {
					foundResponse = true
					fmt.Println("✓✓ 这是我们的响应！\n")

					// 解析数据
					dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
					response := (*simconnect.MobiFlightResponse)(dataPtr)

					// 打印原始数据
					fmt.Printf("原始数据 (前 50 字节): %v\n", response.Data[:50])

					str := simconnect.ParseResponseString(response)
					fmt.Printf("字符串数据: '%s'\n", str)

					if str != "" {
						value, err := simconnect.ParseResponseFloat(response)
						if err == nil {
							fmt.Printf("\n✓✓✓ 成功读取 L:%s = %.6f\n", testVar, value)
						} else {
							fmt.Printf("解析为浮点数失败: %v\n", err)
						}
					}
				}
			}
		}
	}

TestComplete:
	fmt.Println("\n=== 测试完成 ===")

	if foundResponse {
		fmt.Println("\n🎉 成功！MobiFlight WASM 模块工作正常！")
		fmt.Println("\n您现在可以：")
		fmt.Println("1. 使用 SetLVar() 设置飞机参数")
		fmt.Println("2. 使用 GetLVar() 读取飞机参数")
		fmt.Println("3. 查看 main.go 中的完整示例")
	} else {
		fmt.Println("\n⚠ 未收到响应")
		fmt.Println("可能的原因：")
		fmt.Println("1. 当前飞机不支持这个 LVar")
		fmt.Println("2. WASM 模块需要更多时间处理")
		fmt.Println("3. 命令格式可能需要调整")
	}

	fmt.Println()
}
