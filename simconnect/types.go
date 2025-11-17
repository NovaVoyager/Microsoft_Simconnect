package simconnect

// HRESULT 是 Windows API 的返回值类型
type HRESULT uintptr

// SimConnect 接收消息基础结构
type SIMCONNECT_RECV struct {
	DwSize    uint32
	DwVersion uint32
	DwID      uint32
}

// 客户端数据接收结构
type SIMCONNECT_RECV_CLIENT_DATA struct {
	SIMCONNECT_RECV
	RequestID   uint32
	ObjectID    uint32
	DefineID    uint32
	Flags       uint32
	EntryNumber uint32
	OutOf       uint32
	DefineCount uint32
}
