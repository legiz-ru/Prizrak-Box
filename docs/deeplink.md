# Deeplink Profile Import

Prizrak-Box supports importing profiles via deeplink URLs, allowing users to easily add subscriptions from external sources.

## URL Scheme

The deeplink uses the custom protocol `prizrak-box://` with the following format:

```
prizrak-box://install-config?url=SUBSCRIPTION_URL[&name=PROFILE_NAME]
```

### Parameters

- `url` (required): The subscription URL to import
- `name` (optional): Custom name for the imported profile

### Examples

1. **Basic import:**
   ```
   prizrak-box://install-config?url=https://sub.example.com/username
   ```

2. **Import with custom name:**
   ```
   prizrak-box://install-config?url=https://sub.example.com/username&name=MyProfile
   ```

3. **Parameters in different order:**
   ```
   prizrak-box://install-config?name=TestProfile&url=https://another.example.com/config
   ```

## Supported Content Types

The deeplink supports the same content types as manual profile import:

- Subscription URLs (HTTP/HTTPS)
- Share links (vmess://, vless://, ss://, etc.)
- Base64 encoded configurations
- YAML configurations
- JSON configurations

## Implementation Details

### Cross-Platform Support

- **Windows**: Registry entries are automatically created during installation
- **macOS**: Protocol handler is registered via `LSURLTypes` in Info.plist
- **Linux**: MIME type `x-scheme-handler/prizrak-box` is registered

### Protocol Registration

The protocol is registered through:

1. **Electron Main Process**: `app.setAsDefaultProtocolClient('prizrak-box')`
2. **Package Configuration**: Protocol schemes in `forge.config.ts`
3. **Platform-specific entries**: Registry/desktop files during installation

### Error Handling

The implementation includes comprehensive error handling:

- Invalid URL format validation
- Network connection error handling
- Profile parsing error messages
- User-friendly error notifications

### Security Considerations

- Only HTTP/HTTPS URLs are accepted for subscriptions
- URL validation prevents malicious input
- No arbitrary code execution from deeplinks
- User confirmation may be added for security-sensitive operations

## Usage

1. User clicks a deeplink from a webpage or application
2. Operating system launches Prizrak-Box (or brings it to focus)
3. Application parses the deeplink parameters
4. Profile import process begins automatically
5. User receives success/error feedback
6. New profile appears in the profiles list

## Testing

Test files are available in `/tmp/`:
- `deeplink-test.html`: Interactive test page with clickable links
- `test-deeplink.sh`: Command-line test script

## References

This implementation follows similar patterns from other VPN clients:

1. **Sparkle**: `sparkle://install-config?name=USERNAME&url=SUBURL`
2. **Clash Verge Rev**: `clash://install-config?url=<encoded_url>`
3. **FlClash**: `flclash://install-config?url=SUBURL`

Prizrak-Box uses: `prizrak-box://install-config?url=SUBURL&name=NAME`