package simconnect

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unsafe"
)

// MobiFlight WASM 模块常量
const (
	// 客户端数据区域名称
	MOBIFLIGHT_CLIENT_DATA_NAME_COMMAND  = "MobiFlight.Command"
	MOBIFLIGHT_CLIENT_DATA_NAME_RESPONSE = "MobiFlight.Response"
	MOBIFLIGHT_CLIENT_DATA_NAME_SIMVAR   = "MobiFlight.LVars"

	// 客户端数据 ID
	MOBIFLIGHT_CLIENT_DATA_ID_COMMAND  = 0x00004242
	MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE = 0x00004243
	MOBIFLIGHT_CLIENT_DATA_ID_SIMVAR   = 0x00004244

	// 定义 ID
	MOBIFLIGHT_DEFINE_ID_COMMAND  = 0x10000
	MOBIFLIGHT_DEFINE_ID_RESPONSE = 0x10001
	MOBIFLIGHT_DEFINE_ID_SIMVAR   = 0x10002

	// 请求 ID
	MOBIFLIGHT_REQUEST_ID_RESPONSE = 0x20000
	MOBIFLIGHT_REQUEST_ID_SIMVAR   = 0x20001

	// 数据大小
	MOBIFLIGHT_MESSAGE_SIZE = 1024
)

// MobiFlightCommand 是发送给 WASM 模块的命令结构
type MobiFlightCommand struct {
	Code [1024]byte
}

// MobiFlightResponse 是从 WASM 模块接收的响应结构
type MobiFlightResponse struct {
	Data [1024]byte
}

// MobiFlightClient 是 MobiFlight WASM 模块的客户端封装
type MobiFlightClient struct {
	sc *SimConnect
}

// NewMobiFlightClient 创建一个新的 MobiFlight 客户端
func NewMobiFlightClient(sc *SimConnect) (*MobiFlightClient, error) {
	client := &MobiFlightClient{sc: sc}
	if err := client.initialize(); err != nil {
		return nil, err
	}
	return client, nil
}

// initialize 初始化 MobiFlight 客户端数据区域
func (m *MobiFlightClient) initialize() error {
	// 映射命令数据区域
	if err := m.sc.MapClientDataNameToID(MOBIFLIGHT_CLIENT_DATA_NAME_COMMAND, MOBIFLIGHT_CLIENT_DATA_ID_COMMAND); err != nil {
		return fmt.Errorf("failed to map command data area: %v", err)
	}

	// 映射响应数据区域
	if err := m.sc.MapClientDataNameToID(MOBIFLIGHT_CLIENT_DATA_NAME_RESPONSE, MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE); err != nil {
		return fmt.Errorf("failed to map response data area: %v", err)
	}

	// 注意：MobiFlight WASM 模块会自动创建所有数据区域
	// 我们不需要调用 CreateClientData

	// 定义命令数据结构
	if err := m.sc.AddToClientDataDefinition(
		MOBIFLIGHT_DEFINE_ID_COMMAND,
		0,
		MOBIFLIGHT_MESSAGE_SIZE,
		0,
		0,
	); err != nil {
		return fmt.Errorf("failed to define command data: %v", err)
	}

	// 定义响应数据结构
	if err := m.sc.AddToClientDataDefinition(
		MOBIFLIGHT_DEFINE_ID_RESPONSE,
		0,
		MOBIFLIGHT_MESSAGE_SIZE,
		0,
		0,
	); err != nil {
		return fmt.Errorf("failed to define response data: %v", err)
	}

	// 请求接收响应数据
	// 使用 ON_SET + CHANGED 标志，这样只在数据改变时才接收
	if err := m.sc.RequestClientData(
		MOBIFLIGHT_CLIENT_DATA_ID_RESPONSE,
		MOBIFLIGHT_REQUEST_ID_RESPONSE,
		MOBIFLIGHT_DEFINE_ID_RESPONSE,
		SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
		SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED,
		0, 0, 0,
	); err != nil {
		return fmt.Errorf("failed to request response data: %v", err)
	}

	return nil
}

// GetLVar 读取 LVar（本地变量）的值
// varName: LVar 的名称，可以带或不带 "L:" 前缀，例如 "XMLVAR_Switch_AP_MSL_Mode" 或 "L:XMLVAR_Switch_AP_MSL_Mode"
// 返回的值需要通过消息循环调用 ReceiveResponse() 来获取
func (m *MobiFlightClient) GetLVar(varName string) error {
	// 移除可能存在的 L: 前缀，然后重新添加
	varName = strings.TrimPrefix(varName, "L:")
	// RPN 代码格式: (L:VarName)
	command := fmt.Sprintf("(L:%s)", varName)
	return m.sendCommand(command)
}

// SetLVar 设置 LVar（本地变量）的值
// varName: LVar 的名称，可以带或不带 "L:" 前缀，例如 "XMLVAR_Switch_AP_MSL_Mode" 或 "L:XMLVAR_Switch_AP_MSL_Mode"
// value: 要设置的值
// RPN 格式: value (>L:VarName)
func (m *MobiFlightClient) SetLVar(varName string, value float64) error {
	// 移除可能存在的 L: 前缀，然后重新添加
	varName = strings.TrimPrefix(varName, "L:")
	command := fmt.Sprintf("%.6f (>L:%s)", value, varName)
	return m.sendCommand(command)
}

// ExecuteCode 执行 RPN 计算器代码
// code: RPN 计算器代码，例如 "1 (>L:MY_VAR)"
func (m *MobiFlightClient) ExecuteCode(code string) error {
	command := fmt.Sprintf("MF.SimVars.Set.%s", code)
	return m.sendCommand(command)
}

// sendCommand 发送命令到 MobiFlight WASM 模块
func (m *MobiFlightClient) sendCommand(command string) error {
	var cmd MobiFlightCommand

	// 将命令字符串复制到命令结构中
	cmdBytes := []byte(command)
	if len(cmdBytes) >= len(cmd.Code) {
		return fmt.Errorf("command too long: %d bytes (max %d)", len(cmdBytes), len(cmd.Code)-1)
	}

	copy(cmd.Code[:], cmdBytes)

	// 发送命令
	return m.sc.SetClientData(
		MOBIFLIGHT_CLIENT_DATA_ID_COMMAND,
		MOBIFLIGHT_DEFINE_ID_COMMAND,
		SIMCONNECT_CLIENT_DATA_SET_FLAG_DEFAULT,
		0,
		MOBIFLIGHT_MESSAGE_SIZE,
		unsafe.Pointer(&cmd),
	)
}

// ReceiveResponse 接收来自 MobiFlight WASM 模块的响应
// 这个函数需要在消息循环中调用
// 返回 nil, nil 表示当前没有响应消息（不是错误）
// 返回 nil, error 表示真正的错误
// 返回 response, nil 表示成功接收到响应
func (m *MobiFlightClient) ReceiveResponse() (*MobiFlightResponse, error) {
	ppData, _, err := m.sc.GetNextDispatch()
	if err != nil {
		// 真正的错误
		return nil, err
	}

	if ppData == nil {
		// 没有消息，不是错误
		return nil, nil
	}

	recv := (*SIMCONNECT_RECV)(ppData)
	if recv.DwID == SIMCONNECT_RECV_ID_CLIENT_DATA {
		clientData := (*SIMCONNECT_RECV_CLIENT_DATA)(ppData)
		if clientData.RequestID == MOBIFLIGHT_REQUEST_ID_RESPONSE {
			// 数据紧跟在 SIMCONNECT_RECV_CLIENT_DATA 结构之后
			dataPtr := unsafe.Pointer(uintptr(ppData) + unsafe.Sizeof(SIMCONNECT_RECV_CLIENT_DATA{}))
			response := (*MobiFlightResponse)(dataPtr)
			return response, nil
		}
	}

	// 收到消息，但不是我们期望的响应类型
	return nil, nil
}

// ParseResponseFloat 从响应中解析浮点数值
func ParseResponseFloat(response *MobiFlightResponse) (float64, error) {
	// 找到字符串结束位置
	endIdx := 0
	for i, b := range response.Data {
		if b == 0 {
			endIdx = i
			break
		}
	}

	if endIdx == 0 {
		return 0, fmt.Errorf("empty response")
	}

	// 转换为字符串
	str := string(response.Data[:endIdx])
	str = strings.TrimSpace(str)

	// 尝试解析为浮点数
	var value float64
	_, err := fmt.Sscanf(str, "%f", &value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float from response '%s': %v", str, err)
	}

	return value, nil
}

// ParseResponseString 从响应中解析字符串值
func ParseResponseString(response *MobiFlightResponse) string {
	// 找到字符串结束位置
	endIdx := 0
	for i, b := range response.Data {
		if b == 0 {
			endIdx = i
			break
		}
	}

	if endIdx == 0 {
		return ""
	}

	return strings.TrimSpace(string(response.Data[:endIdx]))
}

// LVarValue 包含 LVar 的名称和值
type LVarValue struct {
	Name  string
	Value float64
}

// LVarBatch 用于批量操作 LVar
type LVarBatch struct {
	client *MobiFlightClient
	vars   []LVarValue
}

// NewLVarBatch 创建一个新的 LVar 批量操作对象
func NewLVarBatch(client *MobiFlightClient) *LVarBatch {
	return &LVarBatch{
		client: client,
		vars:   make([]LVarValue, 0),
	}
}

// Add 添加一个 LVar 到批量操作
func (b *LVarBatch) Add(name string, value float64) {
	b.vars = append(b.vars, LVarValue{Name: name, Value: value})
}

// Execute 执行批量设置（将多个 LVar 设置命令合并为一个 RPN 代码）
func (b *LVarBatch) Execute() error {
	if len(b.vars) == 0 {
		return nil
	}

	// 构建 RPN 代码
	var rpnCode strings.Builder
	for i, v := range b.vars {
		if i > 0 {
			rpnCode.WriteString(" ")
		}
		rpnCode.WriteString(fmt.Sprintf("%.6f (>L:%s)", v.Value, v.Name))
	}

	return b.client.ExecuteCode(rpnCode.String())
}

// Helper function: 创建并初始化 MobiFlight 客户端的快捷方式
func SetupMobiFlightClient(clientName string) (*SimConnect, *MobiFlightClient, error) {
	// 加载 SimConnect DLL
	sc, err := LoadDLL()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load SimConnect DLL: %v", err)
	}

	// 打开 SimConnect 连接
	if err := sc.Open(clientName); err != nil {
		return nil, nil, fmt.Errorf("failed to open SimConnect: %v", err)
	}

	// 创建 MobiFlight 客户端
	mf, err := NewMobiFlightClient(sc)
	if err != nil {
		sc.Close()
		return nil, nil, fmt.Errorf("failed to create MobiFlight client: %v", err)
	}

	return sc, mf, nil
}

// 用于二进制数据操作的辅助函数
func writeFloat64(buf []byte, value float64) {
	binary.LittleEndian.PutUint64(buf, uint64(value))
}

func readFloat64(buf []byte) float64 {
	return float64(binary.LittleEndian.Uint64(buf))
}
