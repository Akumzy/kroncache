const ms = require("ms");
const WebSocket = require("ws");
const EventEmitter = require("events");

class Kroncache extends EventEmitter {
  constructor() {
    super();
    this.ws = null;
  }
  connect() {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket("ws://localhost:8080");
      this.ws = ws;
      ws.once("error", reject);
      ws.once("open", () => {
        resolve();
        ws.removeEventListener("open", reject);
        this.boot();
      });
    });
  }
  boot() {
    const ws = this.ws;
    ws.addEventListener("error", (err) => {
      throw err;
    });
    ws.addEventListener("close", (err) => {
      const error = new Error("[Kroncache server close] " + err.reason);
      throw error;
    });
    ws.addEventListener("message", (payload) => {
      if (payload.type === "message") {
        console.log(payload.data);
      }
    });
  }
  set({ key, expire, data }) {
    return new Promise((resolve, reject) => {
      if (this.ws) {
        if (typeof expire !== "number") {
          expire = ms(expire);
        }
        expire = Date.now() + expire;
        this.ws.send(JSON.stringify({ action: "SET" }));
      } else reject("Socket not connected");
    });
  }
}

async function main() {
  try {
    const kron = new Kroncache();
    await kron.connect();
    console.log("connected");
  } catch (error) {
    console.log({ error });
  }
}
main();
