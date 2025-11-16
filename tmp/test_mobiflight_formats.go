package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试不同的 MobiFlight 命令格式
func main() {
	fmt.Println("=== 测试 MobiFlight 命令格式 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("MF 格式测试")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	// 初始化
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		sc.GetNextDispatch()
	}

	sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	sc.MapClientDataNameToID("MobiFlight.Response", 0x00004243)
	sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)
	sc.AddToClientDataDefinition(0x10001, 0, 1024, 0, 0)

	sc.RequestClientData(0x00004243, 0x20000, 0x10001,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
		simconnect.SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED,
		0, 0, 0)

	fmt.Println("✓ 初始化完成\n")

	type MFCommand struct {
		Code [1024]byte
	}

	// 测试不同的命令格式
	testFormats := []struct {
		name    string
		command string
		desc    string
	}{
		{
			"MobiFlight 官方格式 - Get",
			"MF.SimVars.Get.XMLVAR_Switch_AP_MSL_Mode",
			"MobiFlight 文档中的 Get 格式",
		},
		{
			"MobiFlight 官方格式 - Set",
			"MF.SimVars.Set.XMLVAR_Switch_AP_MSL_Mode 1",
			"MobiFlight 文档中的 Set 格式",
		},
		{
			"纯 RPN - Read",
			"(L:XMLVAR_Switch_AP_MSL_Mode)",
			"纯 RPN 读取格式",
		},
		{
			"纯 RPN - Write",
			"1 (>L:XMLVAR_Switch_AP_MSL_Mode)",
			"纯 RPN 写入格式",
		},
		{
			"LVar.Get",
			"LVar.Get XMLVAR_Switch_AP_MSL_Mode",
			"LVar.Get 格式",
		},
		{
			"LVar.Set",
			"LVar.Set XMLVAR_Switch_AP_MSL_Mode 1",
			"LVar.Set 格式",
		},
		{
			"Execute.Calculator.Code",
			"Execute.Calculator.Code (L:XMLVAR_Switch_AP_MSL_Mode)",
			"Execute Calculator Code 格式",
		},
		{
			"MobiFlight.LVars.Get",
			"MobiFlight.LVars.Get XMLVAR_Switch_AP_MSL_Mode",
			"MobiFlight LVars 格式",
		},
	}

	for i, test := range testFormats {
		fmt.Printf("=== 测试 %d: %s ===\n", i+1, test.name)
		fmt.Printf("命令: %s\n", test.command)
		fmt.Printf("说明: %s\n\n", test.desc)

		var cmd MFCommand
		copy(cmd.Code[:], []byte(test.command))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("✗ 发送失败: %v\n\n", err)
			continue
		}

		fmt.Println("✓ 命令已发送")
		fmt.Println("等待 3 秒并监听响应...\n")

		foundClientData := false
		foundResponse := false

		timeout := time.After(3 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)

		msgCount := 0

	Loop:
		for {
			select {
			case <-timeout:
				break Loop

			case <-ticker.C:
				ppData, _, _ := sc.GetNextDispatch()
				if ppData == nil {
					continue
				}

				msgCount++
				recv := (*simconnect.SIMCONNECT_RECV)(ppData)

				fmt.Printf("  [消息 #%d] 类型: %d", msgCount, recv.DwID)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					foundClientData = true
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
					fmt.Printf(" (CLIENT_DATA! RequestID=0x%08X)\n", clientData.RequestID)

					if clientData.RequestID == 0x20000 {
						foundResponse = true
						fmt.Println("  ✓✓✓ 这是我们的响应！")

						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)

						str := simconnect.ParseResponseString(response)
						fmt.Printf("  数据: '%s'\n", str)

						if str != "" {
							value, err := simconnect.ParseResponseFloat(response)
							if err == nil {
								fmt.Printf("  值: %.6f\n", value)
							}
						}

						fmt.Printf("\n🎉 找到工作的命令格式: %s\n", test.name)
						fmt.Printf("命令: %s\n\n", test.command)
						break Loop
					}
				} else {
					fmt.Println()
				}
			}
		}

		ticker.Stop()

		if foundResponse {
			fmt.Println("✅ 这个格式有效！\n")
			fmt.Println("========================================\n")
			break
		} else if foundClientData {
			fmt.Println("⚠ 收到客户端数据消息，但 RequestID 不匹配\n")
		} else if msgCount > 0 {
			fmt.Printf("⚠ 收到 %d 条其他类型的消息，但没有客户端数据\n\n", msgCount)
		} else {
			fmt.Println("⚠ 完全没有收到任何响应\n")
		}

		fmt.Println("----------------------------------------\n")

		// 等待一下再测试下一个
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("=== 测试完成 ===\n")

	fmt.Println("结论:")
	fmt.Println("如果找到了工作的格式，请使用那个格式。")
	fmt.Println("如果所有格式都不工作，可能的原因:")
	fmt.Println("  1. 当前飞机不支持 LVar")
	fmt.Println("  2. MobiFlight WASM 模块版本不兼容")
	fmt.Println("  3. 需要先用 MobiFlight Connector 注册客户端")
	fmt.Println("  4. 需要使用完全不同的通信机制")
	fmt.Println()
}
