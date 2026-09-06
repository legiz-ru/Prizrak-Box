import { ref } from 'vue'

// Shared reactive log level across log views and the WebSocket connection.
// Empty string = use the backend default from mihomo config.
// Not persisted — resets to '' on every app launch.
export const logLevel = ref('')
