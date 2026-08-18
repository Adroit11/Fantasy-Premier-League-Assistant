const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const http = require('http');
const fs = require('fs');

// Simple .env parser for Electron
function loadEnv() {
  const envPaths = [
    path.resolve(__dirname, '../../.env'),
    path.resolve(__dirname, '../.env'),
    path.resolve(__dirname, '.env')
  ];

  for (const p of envPaths) {
    if (fs.existsSync(p)) {
      try {
        const content = fs.readFileSync(p, 'utf-8');
        content.split('\n').forEach(line => {
          const trimmed = line.trim();
          if (!trimmed || trimmed.startsWith('#')) return;
          const [k, ...v] = trimmed.split('=');
          if (k && v.length) {
            const key = k.trim();
            const val = v.join('=').trim().replace(/^["']|["']$/g, '');
            if (!process.env[key]) {
              process.env[key] = val;
            }
          }
        });
        break;
      } catch (e) {
        console.warn('[Electron] Could not read .env file:', e.message);
      }
    }
  }
}

loadEnv();

let mainWindow = null;
let goBackendProcess = null;
const BACKEND_PORT = Number(process.env.BACKEND_PORT || process.env.PORT || 18492);
const BACKEND_URL = process.env.BACKEND_URL || `http://127.0.0.1:${BACKEND_PORT}`;

const ACTION_ROUTES = {
  'connect_team': '/api/v1/team/connect',
  'get_team_overview': '/api/v1/team/overview',
  'get_availability_news': '/api/v1/availability/news',
  'calculate_projections': '/api/v1/projections/calculate',
  'suggest_lineup': '/api/v1/lineup/suggest',
  'suggest_transfers': '/api/v1/transfers/suggest',
};

function checkBackendHealth(retries = 10, delay = 500) {
  return new Promise((resolve, reject) => {
    let attempts = 0;
    const tryConnect = () => {
      attempts++;
      http.get(`${BACKEND_URL}/healthz`, (res) => {
        if (res.statusCode === 200) {
          resolve(true);
        } else if (attempts < retries) {
          setTimeout(tryConnect, delay);
        } else {
          reject(new Error(`Backend health check failed with status: ${res.statusCode}`));
        }
      }).on('error', (err) => {
        if (attempts < retries) {
          setTimeout(tryConnect, delay);
        } else {
          reject(err);
        }
      });
    };
    tryConnect();
  });
}

function startBackend() {
  const binaryName = process.platform === 'win32' ? 'fpl-server.exe' : 'fpl-server';
  const binaryPath = path.resolve(__dirname, '../../backend/bin', binaryName);

  try {
    goBackendProcess = spawn(binaryPath, [`--port=${BACKEND_PORT}`], {
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
    });

    goBackendProcess.stdout.on('data', (data) => {
      console.log(`[Go Core] ${data.toString().trim()}`);
    });

    goBackendProcess.stderr.on('data', (data) => {
      console.error(`[Go Core Error] ${data.toString().trim()}`);
    });

    goBackendProcess.on('exit', (code) => {
      console.log(`[Go Core] Process exited with code ${code}`);
    });
  } catch (err) {
    console.warn(`[Electron] Could not spawn local binary at ${binaryPath}:`, err.message);
  }
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1100,
    minHeight: 700,
    backgroundColor: '#0b0712',
    title: 'FPL Assistant 2026/2027',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  // In development, load Vite dev server or local dist
  if (process.env.VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html')).catch(() => {
      mainWindow.loadFile(path.join(__dirname, '../index.html'));
    });
  }
}

// IPC Action Bridge
ipcMain.handle('fpl:action', async (_event, { actionName, payload }) => {
  const route = ACTION_ROUTES[actionName];
  if (!route) {
    throw new Error(`Unknown action: ${actionName}`);
  }

  const postData = JSON.stringify(payload || {});
  return new Promise((resolve, reject) => {
    const req = http.request(`${BACKEND_URL}${route}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
      },
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          if (res.statusCode >= 400) {
            reject(new Error(parsed.message || parsed.error || `HTTP ${res.statusCode}`));
          } else {
            resolve(parsed);
          }
        } catch (e) {
          reject(new Error(`Failed to parse backend response: ${data}`));
        }
      });
    });

    req.on('error', (err) => {
      reject(new Error(`Backend connection error: ${err.message}`));
    });

    req.write(postData);
    req.end();
  });
});

ipcMain.handle('app:status', async () => {
  try {
    await checkBackendHealth(2, 200);
    return { online: true, backendUrl: BACKEND_URL };
  } catch (e) {
    return { online: false, error: e.message };
  }
});

app.whenReady().then(async () => {
  startBackend();
  try {
    await checkBackendHealth();
    console.log('[Electron] Connected to Go Backend core successfully.');
  } catch (e) {
    console.warn('[Electron] Backend healthcheck warning:', e.message);
  }
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (goBackendProcess) {
    goBackendProcess.kill('SIGTERM');
  }
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  if (goBackendProcess) {
    goBackendProcess.kill('SIGTERM');
  }
});
