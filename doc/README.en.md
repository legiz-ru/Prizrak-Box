<div align="center">
  <img src="../build/appicon.png" style="width:160px" alt="Prizrak-Box"/>
  <h1>Prizrak-Box</h1>
  <p>A simple desktop client for Mihomo</p>
</div>

## Download

[Download the App](https://github.com/legiz-ru/Prizrak-Box/releases)

## Features

- Supports local HTTP/HTTPS/SOCKS proxies
- Supports Vmess, Vless, Shadowsocks, Trojan, Tuic, Hysteria, Hysteria2, Wireguard, and Mieru protocols
- Supports parsing of share links, subscription links, Base64 format, and YAML format
- Built-in subscription converter to convert various subscription types into Mihomo-compatible configurations
- Automatically adds minimal rule groups to unruly subscriptions
- DNS overwrite option to prevent DNS leaks
- Unified rules and group settings for all subscriptions
- Supports TUN mode

## Supported Platforms

- Windows 10/11 (AMD64 / ARM64)
- macOS 11.0+ (AMD64 / ARM64)
- Linux (AMD64 / ARM64)

## How to Enable TUN Mode

- Go to `Settings` → `Enable Authorization` → Restart the app → When the authorization prompt appears, grant
  permission → TUN mode can then be enabled in the app

## Deeplink Profile Import

Prizrak-Box supports importing profiles via deeplink URLs, allowing users to easily add subscriptions from external sources.

### URL Scheme

The deeplink uses the custom protocol `prizrak-box://` with the following format:

```
prizrak-box://install-config?url=SUBSCRIPTION_URL
```

### Parameters

- `url` (required): The subscription URL to import

### Examples

1. **Basic import:**
   ```
   prizrak-box://install-config?url=https://sub.example.com/username
   ```

2. **Import from different providers:**
   ```
   prizrak-box://install-config?url=https://another.provider.com/config
   ```

### Supported Content Types

The deeplink supports all content types that manual profile import supports:

- Subscription URLs (HTTP/HTTPS)
- Share links (vmess://, vless://, ss://, etc.)
- Base64 encoded configurations  
- YAML configurations
- JSON configurations

### Usage

1. User clicks a deeplink from a webpage or application
2. Operating system launches Prizrak-Box (or brings it to focus)  
3. Application automatically imports the profile
4. User receives success/error feedback
5. New profile appears in the profiles list

## Note: Px Requires Network Access

- When prompted, click "Allow" to grant network access

## Common macOS Issues

- See [mac.md](mac/mac.md)

## Major Improvements in the Latest Version

1. Redesigned interface with support for theme switching, language switching, and drag-and-drop import
2. Search bar at the top to quickly switch between nodes in the current configuration
3. Added support for minimizing to system tray
4. Unified rule templates:
    - Simple groups for lightweight users
    - Multi-region groups
    - Full rule groups for advanced users
5. Web scraping and import/export modules from version 0.2 are not yet included

## Todo / Future Plans

- Web scraping module
- Import/export module
- Bug fixes

## Preview

| Tab      | New Interface with Different Themes |
|----------|-------------------------------------|
| Home     | ![General](img/home.png)            |
| Settings | ![Setting](img/setting.png)         |
| Proxies  | ![Proxies](img/proxies.png)         |
| Profiles | ![Profiles](img/profiles.png)       |
