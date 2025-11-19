chrome.runtime.onInstalled.addListener(() => {
  console.log("s0f extension installed");
});

chrome.action.onClicked.addListener(async (tab) => {
  if (!tab.url) {
    return;
  }
  console.log("[scaffold] would send add_bookmark for", tab.url);
});
