# s0f macOS Client Scaffold

This Swift Package hosts the starter SwiftUI app. Use `swift build` or open via Xcode:

```bash
cd ui/macos
swift build
```

The app currently renders placeholder content; upcoming work will embed an IPC client that connects to the Go daemon over Unix sockets and subscribes to `tree_changed` events.
