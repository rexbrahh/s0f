const HOST_NAME = "com.example.bmd";
let nativePort: chrome.runtime.Port | null = null;

function ensurePort() {
  if (nativePort) {
    return nativePort;
  }
  nativePort = chrome.runtime.connectNative(HOST_NAME);
  nativePort.onMessage.addListener((msg) => {
    console.log("[s0f native]", msg);
  });
  nativePort.onDisconnect.addListener(() => {
    console.warn("Native port disconnected");
    nativePort = null;
  });
  return nativePort;
}

function sendNative(type: string, params: Record<string, unknown> = {}) {
  const port = ensurePort();
  const message = {
    id: `chrome-${Date.now()}`,
    type,
    params,
  };
  port.postMessage(message);
}

chrome.runtime.onInstalled.addListener(() => {
  console.log("s0f extension installed");
  sendNative("ping");
});

chrome.action.onClicked.addListener(async (tab) => {
  if (!tab.url || !tab.title) {
    return;
  }
  sendNative("apply_ops", {
    ops: [
      {
        type: "add_bookmark",
        parentId: "root",
        title: tab.title,
        url: tab.url,
      },
    ],
  });
});

chrome.runtime.onStartup.addListener(() => {
  ensurePort();
});
