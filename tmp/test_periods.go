package main

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
)

// 测试不同的请求周期设置
func main() {
	fmt.Println("=== 测试不同的 Period 设置 ===\n")

	sc, err := simconnect.LoadDLL()
	if err != nil {
		log.Fatal(err)
	}

	err = sc.Open("Period 测试")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	// 初始化客户端数据区域
	sc.MapClientDataNameToID("MobiFlight.Command", 0x00004242)
	sc.MapClientDataNameToID("MobiFlight.Response", 0x00004243)
	sc.CreateClientData(0x00004242, 1024, 0)
	sc.AddToClientDataDefinition(0x10000, 0, 1024, 0, 0)
	sc.AddToClientDataDefinition(0x10001, 0, 1024, 0, 0)

	fmt.Println("✓ 初始化完成\n")

	// 测试不同的 Period 设置
	periods := []struct {
		name  string
		value uint32
	}{
		{"ON_SET", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET},
		{"ONCE", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_ONCE},
		{"VISUAL_FRAME", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_VISUAL_FRAME},
		{"NEVER", simconnect.SIMCONNECT_CLIENT_DATA_PERIOD_NEVER},
	}

	for i, period := range periods {
		fmt.Printf("测试 %d: Period = %s (0x%X)\n", i+1, period.name, period.value)

		requestID := uint32(0x20000 + i)

		// 请求客户端数据
		err = sc.RequestClientData(
			0x00004243,   // Response 区域 ID
			requestID,    // 唯一的请求 ID
			0x10001,      // 定义 ID
			period.value, // 不同的 Period
			0,            // Flags
			0, 0, 0,
		)

		if err != nil {
			fmt.Printf("  ✗ RequestClientData 失败: %v\n\n", err)
			continue
		}

		fmt.Println("  ✓ RequestClientData 成功")

		// 发送测试命令
		type Cmd struct {
			Code [1024]byte
		}
		var cmd Cmd
		cmdStr := "(L:XMLVAR_Switch_AP_MSL_Mode)"
		copy(cmd.Code[:], []byte(cmdStr))

		err = sc.SetClientData(0x00004242, 0x10000, 0, 0, 1024, unsafe.Pointer(&cmd))
		if err != nil {
			fmt.Printf("  ✗ SetClientData 失败: %v\n\n", err)
			continue
		}

		fmt.Println("  ✓ 命令已发送，等待响应...")

		// 尝试接收响应
		foundResponse := false
		for j := 0; j < 20; j++ {
			ppData, _, _ := sc.GetNextDispatch()

			if ppData == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			recv := (*simconnect.SIMCONNECT_RECV)(ppData)

			if recv.DwID == simconnect.SIMCONNECT_RECV_ID_CLIENT_DATA {
				clientData := (*simconnect.SIMCONNECT_RECV_CLIENT_DATA)(ppData)

				if clientData.RequestID == requestID {
					foundResponse = true
					fmt.Printf("  ✓✓ 收到响应！(Request ID: %d)\n", requestID)

					dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(simconnect.SIMCONNECT_RECV_CLIENT_DATA{}))
					response := (*simconnect.MobiFlightResponse)(dataPtr)
					str := simconnect.ParseResponseString(response)
					fmt.Printf("  数据: '%s'\n", str)
					break
				} else {
					fmt.Printf("  收到客户端数据，但 Request ID 不匹配: %d (期望 %d)\n", clientData.RequestID, requestID)
				}
			}
		}

		if !foundResponse {
			fmt.Println("  ⚠ 未收到响应")
		}

		fmt.Println()

		// 等待一会儿再测试下一个
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("=== 测试完成 ===")
	fmt.Println("\n如果某个 Period 设置有效，请在代码中使用该设置。")
}
