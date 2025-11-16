package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试客户端注册流程
func main() {
	fmt.Println("=== 测试 MobiFlight 客户端注册 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("注册测试")
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

	fmt.Println("✓ 基础初始化完成\n")

	type MFCommand struct {
		Code [1024]byte
	}

	// 步骤 1: 尝试注册客户端
	fmt.Println("=== 步骤 1: 注册客户端 ===\n")

	registrationCommands := []string{
		"MF.Clients.Add.GoClient",
		"MF.Config.MAX_VARS_PER_FRAME.SET 30",
		"MF.SimVars.Clear",
	}

	for _, regCmd := range registrationCommands {
		fmt.Printf("发送注册命令: %s\n", regCmd)

		var cmd MFCommand
		copy(cmd.Code[:], []byte(regCmd))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
		} else {
			fmt.Println("  ✓ 发送成功")
		}

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println()

	// 步骤 2: 设置请求接收响应
	fmt.Println("=== 步骤 2: 请求接收响应 ===\n")

	err = sc.RequestClientData(0x00004243, 0x20000, 0x10001,
		simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
		simconnect.SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED,
		0, 0, 0)

	if err != nil {
		fmt.Printf("✗ RequestClientData 失败: %v\n\n", err)
	} else {
		fmt.Println("✓ RequestClientData 成功\n")
	}

	time.Sleep(500 * time.Millisecond)

	// 步骤 3: 添加一个 SimVar 订阅
	fmt.Println("=== 步骤 3: 添加 SimVar 订阅 ===\n")

	addVarCommands := []string{
		"MF.SimVars.Add.XMLVAR_Switch_AP_MSL_Mode.number",
		"MF.SimVars.Add.0.L:XMLVAR_Switch_AP_MSL_Mode.number",
	}

	for _, addCmd := range addVarCommands {
		fmt.Printf("添加订阅: %s\n", addCmd)

		var cmd MFCommand
		copy(cmd.Code[:], []byte(addCmd))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
		} else {
			fmt.Println("  ✓ 发送成功")

			// 等待并检查响应
			time.Sleep(1 * time.Second)

			fmt.Println("  检查响应...")
			foundResponse := false

			for j := 0; j < 10; j++ {
				ppData, _, _ := sc.GetNextDispatch()
				if ppData == nil {
					continue
				}

				recv := (*simconnect.SIMCONNECT_RECV)(ppData)
				fmt.Printf("    收到消息类型: %d\n", recv.DwID)

				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)
					fmt.Printf("    ✓✓ CLIENT_DATA! RequestID=0x%08X\n", clientData.RequestID)

					if clientData.RequestID == 0x20000 {
						foundResponse = true
						fmt.Println("    ✓✓✓ 收到响应！")

						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)

						str := simconnect.ParseResponseString(response)
						fmt.Printf("    数据: '%s'\n", str)
					}
				}
			}

			if !foundResponse {
				fmt.Println("    ⚠ 未收到响应")
			}
		}

		fmt.Println()
	}

	// 步骤 4: 测试标准的 Get 命令
	fmt.Println("=== 步骤 4: 测试 Get 命令 ===\n")

	getCommands := []string{
		"MF.SimVars.Get.0",
		"MF.SimVars.Read",
		"MF.DummyCallback",
	}

	for _, getCmd := range getCommands {
		fmt.Printf("发送: %s\n", getCmd)

		var cmd MFCommand
		copy(cmd.Code[:], []byte(getCmd))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
		} else {
			fmt.Println("  ✓ 发送成功")

			time.Sleep(1 * time.Second)

			fmt.Println("  检查响应...")
			foundResponse := false

			for j := 0; j < 10; j++ {
				ppData, _, _ := sc.GetNextDispatch()
				if ppData == nil {
					continue
				}

				recv := (*simconnect.SIMCONNECT_RECV)(ppData)
				if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
					clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

					if clientData.RequestID == 0x20000 {
						foundResponse = true
						fmt.Println("  ✓✓✓ 收到响应！")

						dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
						response := (*simconnect.MobiFlightResponse)(dataPtr)

						str := simconnect.ParseResponseString(response)
						fmt.Printf("  数据: '%s'\n", str)

						fmt.Printf("\n🎉 成功！使用的流程:\n")
						fmt.Println("  1. 注册客户端")
						fmt.Println("  2. 添加订阅")
						fmt.Printf("  3. 使用 Get 命令: %s\n", getCmd)
						goto TestComplete
					}
				}
			}

			if !foundResponse {
				fmt.Println("  ⚠ 未收到响应")
			}
		}

		fmt.Println()
	}

TestComplete:
	fmt.Println("\n=== 测试完成 ===\n")

	fmt.Println("诊断:")
	fmt.Println("如果收到了响应:")
	fmt.Println("  → 需要先注册客户端并添加订阅")
	fmt.Println("  → MobiFlight 使用订阅模式，不是直接查询")
	fmt.Println()
	fmt.Println("如果仍未收到响应:")
	fmt.Println("  → 可能需要在 MobiFlight Connector 运行的情况下测试")
	fmt.Println("  → 或者需要使用完全不同的通信方式")
	fmt.Println()
}
