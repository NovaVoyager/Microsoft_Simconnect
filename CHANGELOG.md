# 更新日志

## 2025-01-15 - v0.2.0

### 🔧 重要修复

#### 1. 修复消息接收机制

**问题：**
- `GetNextDispatch()` 在没有消息时返回错误，导致 `ReceiveResponse()` 无法正常工作
- 用户收不到 LVar 的响应数据

**修复：**
- 更新 `GetNextDispatch()` 正确处理 SimConnect 返回值：
  - `HRESULT = 0` (S_OK) → 成功接收到消息
  - `HRESULT = 1` (S_FALSE) → 没有消息（不是错误）
  - 其他非零值 → 真正的错误

- 更新 `ReceiveResponse()` 返回值语义：
  - `response != nil, err == nil` → 成功收到响应
  - `response == nil, err == nil` → 没有消息（继续轮询）
  - `response == nil, err != nil` → 发生错误

**代码示例：**
```go
// 新的正确用法
response, err := mf.ReceiveResponse()
if err != nil {
    // 真正的错误
    log.Fatal(err)
}
if response == nil {
    // 没有消息，继续轮询
    continue
}
// 处理响应
value, _ := simconnect.ParseResponseFloat(response)
```

#### 2. 修复客户端数据区域初始化

**问题：**
- `EXCEPTION_OUT_OF_BOUNDS` 错误
- 尝试向未创建的客户端数据区域写入数据

**修复：**
- 在 `initialize()` 函数中添加 `CreateClientData()` 调用
- 即使 WASM 模块已创建数据区域，调用也不会失败

#### 3. 修复 GetLVar 命令格式

**问题：**
- 使用了错误的命令格式 `MF.SimVars.Get.VarName`

**修复：**
- 改为正确的 RPN 格式：`(L:VarName)`
- 添加 LVar 名称前缀处理（自动移除和添加 `L:` 前缀）

**代码变更：**
```go
// 之前（错误）
command := fmt.Sprintf("MF.SimVars.Get.%s", varName)

// 现在（正确）
varName = strings.TrimPrefix(varName, "L:")
command := fmt.Sprintf("(L:%s)", varName)
```

### ✨ 新增功能

#### 1. 调试文档

新增 `DEBUG_RESPONSES.md`，提供：
- 完整的调试步骤
- 常见问题和解决方案
- 使用 MobiFlight Connector 验证的方法
- SimConnect 日志启用方法
- 最小复现代码示例

#### 2. 连接测试程序

新增 `test_connection.go`，提供：
- 自动测试所有关键步骤
- 详细的错误提示和建议
- 帮助用户快速诊断问题

### 📝 文档更新

1. **README.md**
   - 添加快速开始指南
   - 添加调试文档链接
   - 更新故障排除部分

2. **MOBIFLIGHT_GUIDE.md**
   - 更新 `ReceiveResponse()` API 文档
   - 更新所有示例代码
   - 添加返回值说明

3. **TROUBLESHOOTING.md**
   - 新增详细的故障排除步骤
   - MobiFlight WASM 模块安装指南
   - SimConnect 错误码参考

### 🔄 代码示例更新

更新了所有示例代码以反映新的 API：
- `main.go` - 基础示例
- `examples/advanced_example.go` - 高级示例
- `test_connection.go` - 测试程序

### ⚠️ 破坏性变更

#### `ReceiveResponse()` 行为变更

**之前：**
```go
response, err := mf.ReceiveResponse()
if err != nil {
    // 没有消息时会返回错误
    continue
}
```

**现在：**
```go
response, err := mf.ReceiveResponse()
if err != nil {
    // 只有真正的错误才会返回 error
    log.Fatal(err)
}
if response == nil {
    // 没有消息（正常情况）
    continue
}
```

**迁移指南：**
所有调用 `ReceiveResponse()` 的代码都需要更新检查逻辑：

```go
// 旧代码
response, err := mf.ReceiveResponse()
if err != nil {
    continue  // 忽略错误
}
if response != nil {
    // 处理响应
}

// 新代码
response, err := mf.ReceiveResponse()
if err != nil {
    return err  // 处理真正的错误
}
if response == nil {
    continue  // 没有消息
}
// 处理响应
```

---

## 2025-01-14 - v0.1.0

### 初始版本

- SimConnect 基础封装
- MobiFlight WASM 模块支持
- LVar 读写功能
- 批量操作支持
- RPN 代码执行
- 基础文档和示例

---

## 后续计划

### v0.3.0（计划中）

- [ ] 改进消息处理机制
- [ ] 添加异步回调支持
- [ ] 支持更多 SimConnect 数据类型
- [ ] 性能优化
- [ ] 添加单元测试

### v0.4.0（计划中）

- [ ] 支持 SimConnect 事件
- [ ] 支持标准 SimVar（不需要 MobiFlight）
- [ ] 添加连接池管理
- [ ] 改进错误处理和恢复

---

## 贡献者

感谢所有贡献者！

如果您发现问题或有改进建议，请：
1. 查看 [TROUBLESHOOTING.md](TROUBLESHOOTING.md) 和 [DEBUG_RESPONSES.md](DEBUG_RESPONSES.md)
2. 创建 GitHub Issue
3. 提交 Pull Request

---

## 许可证

MIT License - 详见 [LICENSE](LICENSE)
