const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("colignDesktop", {
  platform: process.platform,
  isDesktop: true,
  showNotification: (title, body) => {
    ipcRenderer.send("show-notification", { title, body });
  },
});

// Intercept web Notification API → native Electron notification
const OriginalNotification = window.Notification;

class ElectronNotification {
  constructor(title, options = {}) {
    ipcRenderer.send("show-notification", {
      title,
      body: options.body || "",
      url: options.data?.url || "",
    });
  }

  static get permission() {
    return "granted";
  }

  static requestPermission() {
    return Promise.resolve("granted");
  }
}

window.Notification = ElectronNotification;

// Add desktop class to html element for CSS targeting
window.addEventListener("DOMContentLoaded", () => {
  document.documentElement.classList.add("desktop-app");
});
