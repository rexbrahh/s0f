# s0f Chrome Extension Scaffold

## Setup

```bash
cd ui/chrome
npm install
npm run build
```

Load `ui/chrome/dist` as an unpacked extension in Chrome. The build step copies `public/manifest.json` and compiles `src/background.ts` into `dist/background.js`.

Set up the Native Messaging host by running `scripts/install-native-messaging.sh /path/to/bin/bmd-bridge <extension-id>` and ensure the daemon socket is running. The background service worker will connect to the host `com.example.bmd`, send `ping` on install, and push the active tab as a bookmark to the daemon when the action button is clicked.
