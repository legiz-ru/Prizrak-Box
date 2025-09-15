<div align="center">
  <img src="../build/appicon.png" style="width:160px" alt="Prizrak-Box"/>
  <h1>Prizrak-Box</h1>
  <p>一个简易的 Mihomo 桌面客户端</p>
</div>

## 下载地址

[下载APP](https://github.com/legiz-ru/Prizrak-Box/releases)

## 功能特点

- 支持本地 HTTP/HTTPS/SOCKS 代理
- 支持 Vmess, Vless, Shadowsocks, Trojan, Tuic, Hysteria, Hysteria2, Wireguard, Mieru 协议
- 支持分享链接、订阅链接、Base64 格式、Yaml 格式的数据解析
- 内置订阅转换，可将各种订阅转换为 mihomo 配置
- 对无规则订阅自动添加极简规则分组
- 开启 DNS 覆写可防止 DNS 泄露
- 支持统一所有订阅的规则和分组
- 支持 TUN 模式

## 支持的系统平台

- Windows 10/11 AMD64/ARM64
- macOS 11.0+ AMD64/ARM64
- Linux AMD64/ARM64

## 如何开启 TUN

- 设置 → 开启授权 → 重启软件 → 弹出授权框 → 完成授权
- 进入软件后即可开启 TUN 模式

## 深度链接配置导入

Prizrak-Box 支持通过深度链接 URL 导入配置，让用户可以轻松地从外部来源添加订阅。

### URL 格式

深度链接使用自定义协议 `prizrak-box://`，格式如下：

```
prizrak-box://install-config?url=SUBSCRIPTION_URL
```

### 参数

- `url`（必需）：要导入的订阅 URL

### 示例

1. **基础导入：**
   ```
   prizrak-box://install-config?url=https://sub.example.com/username
   ```

2. **从不同提供商导入：**
   ```
   prizrak-box://install-config?url=https://another.provider.com/config
   ```

### 支持的内容类型

深度链接支持手动导入配置支持的所有内容类型：

- 订阅 URL（HTTP/HTTPS）
- 分享链接（vmess://、vless://、ss:// 等）
- Base64 编码配置
- YAML 配置
- JSON 配置

### 使用方法

1. 用户从网页或应用程序点击深度链接
2. 操作系统启动 Prizrak-Box（或将其置于前台）
3. 应用程序自动导入配置
4. 用户收到成功/错误反馈
5. 新配置出现在配置列表中

## 提示 Px 需要网络接入

- 点击 “允许” 即可

## macOS 常见问题汇总

- [mac.md](mac/mac.md)

## 新版主要改进

1. 界面改版：支持背景切换、语言切换、拖拽导入
2. 顶部搜索当前配置节点，快速切换
3. 增加最小化到托盘功能
4. 统一规则模板：简约分组、多国别分组、全分组
5. 暂未迁移 v0.2 版本的爬取模块、导入导出模块

## Todo 未来计划

- 爬取模块
- 导入导出模块
- Bug 修复

## 预览

| 页面 | 界面预览                          |
|----|-------------------------------|
| 首页 | ![General](img/home.png)      |
| 设置 | ![Setting](img/setting.png)   |
| 代理 | ![Proxies](img/proxies.png)   |
| 订阅 | ![Profiles](img/profiles.png) |
