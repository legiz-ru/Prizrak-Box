<div align="center">

  <img src="frontend/public/tray.png" width="160px" alt="Prizrak-Box"/>

  <h1>Prizrak-Box</h1>

  <p>🌈 A simple desktop client for <strong>Mihomo</strong></p>
  <p>✨ 一个简易的 <strong>Mihomo</strong> 桌面客户端</p>
  <p>✨ Простой настольный клиент для <strong>Mihomo</strong></p>

  <p>
    🇨🇳 <a href="doc/README.zh-CN.md">简体中文</a> | 🇺🇸 <a href="doc/README.en.md">English</a> | 🇷🇺 <a href="doc/README.ru.md">Русский</a>
  </p>

</div>

---

## 📦 Project Overview | 项目简介 | Обзор проекта

**Prizrak-Box** is a lightweight and user-friendly cross-platform client (fork of [Pandora-Box](https://github.com/snakem982/Pandora-Box)) for [Mihomo](https://github.com/MetaCubeX/mihomo), supporting multiple proxy protocols, automatic rule grouping, TUN mode, and functionality relevant for Russia.  
It is designed for both casual and advanced users to easily manage and convert proxy subscriptions.

**Prizrak-Box** 是一个跨平台的轻量桌面客户端（[Pandora-Box](https://github.com/snakem982/Pandora-Box) 的分支项目），适配 [Mihomo](https://github.com/MetaCubeX/mihomo) 内核，支持多种代理协议、规则自动分组与 TUN 模式，并具备适用于俄罗斯的相关功能。界面简洁，功能强大，适合轻量与进阶用户使用。

**Prizrak-Box** — это легкий и удобный кроссплатформенный клиент (форк [Pandora-Box](https://github.com/snakem982/Pandora-Box)) для [Mihomo](https://github.com/MetaCubeX/mihomo), поддерживающий различные прокси-протоколы, автоматическую группировку правил, режим TUN и функционал, актуальный для России.  
Он разработан как для обычных пользователей, так и для продвинутых, чтобы облегчить управление и конвертацию подписок прокси.

---

## 📥 Get Started ｜ 快速开始 ｜ Начало работы

👉 [Download the Latest Release / 下载最新版本 / Скачать последнюю версию](https://github.com/legiz-ru/Prizrak-Box/releases)

---

## 🛠 Development ｜ 开发 ｜ Разработка

If you want to contribute or build Prizrak-Box locally, refer to the resources below:  
如果你想参与开发或构建 Prizrak-Box，可以参考以下资源：  
Если вы хотите принять участие в разработке или собрать Prizrak-Box локально, воспользуйтесь следующими ресурсами:

### 🔧 Prerequisites | 前置依赖 | Предварительные требования

- [Node.js](https://nodejs.org/) ≥ 18 (for building the Vue UI)
- [Go](https://go.dev/) ≥ 1.22
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### 🧪 Build Instructions | 构建指南 | Инструкции по сборке

```bash
# Install frontend dependencies
npm --prefix frontend install

# Optional: keep Go modules tidy
go mod tidy

# Run with hot reload
wails dev

# Build distributable binary
wails build
```

---

## 🌐 Language ｜ 语言选择 ｜ Выбор языка

- 🇨🇳 [查看中文文档](doc/README.zh-CN.md)
- 🇺🇸 [View English Documentation](doc/README.en.md)
- 🇷🇺 [Просмотр русской документации](doc/README.ru.md)

---

## 🧭 More Information ｜ 更多信息 ｜ Дополнительная информация

- ✅ [Project Issues](https://github.com/legiz-ru/Prizrak-Box/issues)
- 📄 [License (GPL-3.0)](./LICENSE)

---

📝 This README was generated with the assistance of AI and reviewed by the developer.  
📝 本文档内容由 AI 辅助生成，并由开发者校对。  
📝 Этот README создан при поддержке ИИ и проверен разработчиком.
