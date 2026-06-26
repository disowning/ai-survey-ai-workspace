chrome.runtime.onInstalled.addListener(() => {
  chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true }).catch(() => {
    // Some Chromium builds may not support this yet; the declared side panel still works.
  });
});

chrome.action.onClicked.addListener(async (tab) => {
  if (!tab.windowId) return;

  await chrome.sidePanel.open({ windowId: tab.windowId });
});
