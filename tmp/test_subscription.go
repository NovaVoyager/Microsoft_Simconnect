package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试订阅模式
func main() {
	fmt.Println("=== 测试 MobiFlight 订阅模式 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("订阅模式测试")
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

	sendCommand := func(cmdStr string) {
		var cmd MFCommand
		copy(cmd.Code[:], []byte(cmdStr))
		sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
	}

	checkResponses := func(desc string, duration time.Duration) (bool, string) {
		fmt.Printf("%s (检查 %.1f 秒)...\n", desc, duration.Seconds())

		timeout := time.After(duration)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		msgCount := 0

		for {
			select {
			case <-timeout:
				if msgCount > 0 {
					fmt.Printf("  收到 %d 条消息，但没有客户端数据\n", msgCount)
				} else {
					fmt.Println("  没有收到任何消息")
				}
				return false, ""

			case <-ticker.C:
				ppData, _, _ := sc.GetNextDispatch()
				if ppData == nil {
					continue
				}

				msgCount++
				recv := (*simconnect.SIMCONNECT_RECV)(ppData)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

					if clientData.RequestID == 0x20000 {
						fmt.Println("  ✓✓ 收到响应！")

						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)

						str := simconnect.ParseResponseString(response)
						fmt.Printf("  数据: '%s'\n", str)
						return true, str
					}
				}
			}
		}
	}

	// 完整的订阅流程测试
	fmt.Println("=== 完整订阅流程 ===\n")

	// 1. 清除现有订阅
	fmt.Println("步骤 1: 清除现有订阅")
	sendCommand("MF.SimVars.Clear")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("  ✓ 命令已发送\n")

	// 2. 添加订阅
	fmt.Println("步骤 2: 添加 LVar 订阅")
	subscriptions := []struct {
		id   int
		name string
	}{
		{0, "L:XMLVAR_Switch_AP_MSL_Mode"},
		{1, "L:XMLVAR_Autopilot_Altitude"},
	}

	for _, sub := range subscriptions {
		cmdStr := fmt.Sprintf("MF.SimVars.Add.%d.%s.number", sub.id, sub.name)
		fmt.Printf("  添加订阅 %d: %s\n", sub.id, sub.name)
		sendCommand(cmdStr)
		time.Sleep(300 * time.Millisecond)

		found, _ := checkResponses("    检查响应", 1*time.Second)
		if found {
			fmt.Println("    ✓ 订阅成功\n")
		} else {
			fmt.Println("    ⚠ 未收到确认\n")
		}
	}

	// 3. 请求读取所有订阅
	fmt.Println("步骤 3: 读取所有订阅的值")
	sendCommand("MF.SimVars.Read")
	fmt.Println("  ✓ Read 命令已发送")

	found, data := checkResponses("  等待响应", 2*time.Second)
	if found {
		fmt.Println("\n🎉 成功！MobiFlight 使用订阅模式！")
		fmt.Println("\n工作流程:")
		fmt.Println("  1. MF.SimVars.Clear - 清除订阅")
		fmt.Println("  2. MF.SimVars.Add.<ID>.<LVar>.<type> - 添加订阅")
		fmt.Println("  3. MF.SimVars.Read - 读取所有订阅的值")
		fmt.Printf("\n响应数据: %s\n", data)
	} else {
		fmt.Println()

		// 4. 尝试其他可能的触发命令
		fmt.Println("步骤 4: 尝试其他触发命令")

		triggerCommands := []string{
			"MF.SimVars.Get.0",
			"MF.DummyCallback",
			"MF.Ping",
		}

		success := false
		for _, trigCmd := range triggerCommands {
			fmt.Printf("  发送: %s\n", trigCmd)
			sendCommand(trigCmd)

			found, data := checkResponses("    检查响应", 1*time.Second)
			if found {
				fmt.Printf("\n🎉 成功！触发命令: %s\n", trigCmd)
				fmt.Printf("响应数据: %s\n", data)
				success = true
				break
			}
			fmt.Println()
		}

		if !success {
			// 5. 尝试设置值
			fmt.Println("步骤 5: 尝试设置值")

			setCommands := []string{
				"MF.SimVars.Set.0 1",
				"1 (>L:XMLVAR_Switch_AP_MSL_Mode)",
			}

			for _, setCmd := range setCommands {
				fmt.Printf("  发送: %s\n", setCmd)
				sendCommand(setCmd)

				found, data := checkResponses("    检查响应", 1*time.Second)
				if found {
					fmt.Printf("\n🎉 成功！设置命令: %s\n", setCmd)
					fmt.Printf("响应数据: %s\n", data)
					break
				}
				fmt.Println()
			}
		}
	}
	fmt.Println("\n=== 测试完成 ===\n")

	fmt.Println("如果以上测试仍未成功，请尝试:")
	fmt.Println("  1. 在 MobiFlight Connector 运行的情况下重新测试")
	fmt.Println("  2. 检查当前飞机是否支持 LVar")
	fmt.Println("  3. 查看 MSFS 开发者模式中的 Behavior Debug Window")
	fmt.Println("  4. 检查 MobiFlight WASM 模块的日志")
	fmt.Println()
}
