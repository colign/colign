const { app, BrowserWindow, shell, Tray, Menu, nativeImage, Notification, ipcMain } = require("electron");
const path = require("path");
const Store = require("electron-store");

const APP_URL = "https://app.colign.co";
const PROTOCOL = "colign";

app.setName("Colign");

const store = new Store({
  defaults: {
    windowBounds: { width: 1280, height: 800 },
  },
});

let mainWindow = null;
let tray = null;

function createWindow() {
  const { width, height, x, y } = store.get("windowBounds");

  mainWindow = new BrowserWindow({
    width,
    height,
    x,
    y,
    minWidth: 800,
    minHeight: 600,
    titleBarStyle: "hiddenInset",
    trafficLightPosition: { x: 16, y: 16 },
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  mainWindow.loadURL(APP_URL);

  // Open external links in browser
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (!url.startsWith(APP_URL)) {
      shell.openExternal(url);
      return { action: "deny" };
    }
    return { action: "allow" };
  });

  // Persist window bounds
  mainWindow.on("close", () => {
    const bounds = mainWindow.getBounds();
    store.set("windowBounds", bounds);
  });

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

function createTray() {
  const iconPath = path.join(__dirname, "assets", "tray-icon.png");
  const icon = nativeImage.createFromPath(iconPath).resize({ width: 16, height: 16 });
  tray = new Tray(icon);

  const contextMenu = Menu.buildFromTemplate([
    {
      label: "Open Colign",
      click: () => {
        if (mainWindow) {
          mainWindow.show();
          mainWindow.focus();
        } else {
          createWindow();
        }
      },
    },
    { type: "separator" },
    { label: "Quit", click: () => app.quit() },
  ]);

  tray.setToolTip("Colign");
  tray.setContextMenu(contextMenu);

  tray.on("click", () => {
    if (mainWindow) {
      mainWindow.show();
      mainWindow.focus();
    } else {
      createWindow();
    }
  });
}

function setupDeepLink() {
  if (process.defaultApp) {
    if (process.argv.length >= 2) {
      app.setAsDefaultProtocolClient(PROTOCOL, process.execPath, [path.resolve(process.argv[1])]);
    }
  } else {
    app.setAsDefaultProtocolClient(PROTOCOL);
  }
}

function handleDeepLink(url) {
  if (!url) return;
  const parsed = new URL(url);
  const appPath = parsed.host + parsed.pathname + parsed.search;

  // colign://auth/verified → go to login page
  const targetPath = appPath === "auth/verified" ? "/auth" : `/${appPath}`;

  if (mainWindow) {
    mainWindow.loadURL(`${APP_URL}${targetPath}`);
    mainWindow.show();
    mainWindow.focus();
  }
}

// macOS: single instance + deep link
const gotLock = app.requestSingleInstanceLock();
if (!gotLock) {
  app.quit();
} else {
  app.on("second-instance", (_event, argv) => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
    // Windows/Linux deep link
    const deepUrl = argv.find((arg) => arg.startsWith(`${PROTOCOL}://`));
    if (deepUrl) handleDeepLink(deepUrl);
  });

  app.on("open-url", (_event, url) => {
    handleDeepLink(url);
  });

  app.whenReady().then(() => {
    setupDeepLink();
    createWindow();
    createTray();

    // Native notification bridge
    ipcMain.on("show-notification", (_event, { title, body, url }) => {
      const notification = new Notification({ title, body });
      notification.on("click", () => {
        if (mainWindow) {
          if (url) mainWindow.loadURL(`${APP_URL}${url}`);
          mainWindow.show();
          mainWindow.focus();
        }
      });
      notification.show();
    });

    app.on("activate", () => {
      if (BrowserWindow.getAllWindows().length === 0) {
        createWindow();
      } else if (mainWindow) {
        mainWindow.show();
      }
    });
  });

  app.on("window-all-closed", () => {
    if (process.platform !== "darwin") {
      app.quit();
    }
  });
}
