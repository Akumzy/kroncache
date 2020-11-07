const ms = require("ms");
const WebSocket = require("ws");
const { EventEmitter } = require("events");
const { v4 } = require("uuid");
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
        ws.removeEventListener("error", reject);
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
        /**@type {{action:'SET'|'RESPONSE'|'EXPIRED';data?:string;error?:string;key:string:expire:string;id:string}} */
        let p = JSON.parse(payload.data);

        let data = null;
        if (p.data) {
          data = JSON.parse(p.data);
          data = data.value;
        }
        if (p.action === "EXPIRED") {
          this.emit("expired", { data, expire: p.expire, key: p.key });
        } else {
          this.emit(p.id, p.error, data);
        }
      }
    });
  }
  set({ key, expire, data }) {
    return new Promise((resolve, reject) => {
      if (this.ws) {
        if (typeof expire !== "number") {
          expire = ms(expire);
        }
        expire = new Date(Date.now() + expire);
        const id = v4();
        /**@todo put a timeout */
        this.once(id, (err) => {
          err ? reject(err) : resolve();
        });
        this.ws.send(
          JSON.stringify({
            key,
            action: "SET",
            data: JSON.stringify({ value: data }),
            id,
            expire,
          }),
        );
      } else reject("Socket not connected");
    });
  }
  get(key) {
    return new Promise((resolve, reject) => {
      if (this.ws) {
        const id = v4();
        this.ws.send(
          JSON.stringify({
            key,
            action: "GET",
            id,
          }),
        );
        /**@todo put a timeout */

        this.once(id, (err, data) => {
          err ? reject(err) : resolve(data);
        });
      } else reject("Socket not connected");
    });
  }
  purgeAll() {
    return new Promise((resolve, reject) => {
      if (this.ws) {
        const id = v4();
        this.ws.send(
          JSON.stringify({
            action: "PURGE",
            id,
          }),
        );
        /**@todo put a timeout */

        this.once(id, (err) => {
          err ? reject(err) : resolve();
        });
      } else reject("Socket not connected");
    });
  }
}

async function main() {
  try {
    const kron = new Kroncache();
    await kron.connect();
    // await kron.purgeAll();
    // console.log("Purged");
    kron.addListener("expired", (d) => {
      console.log("Expired: ", d);
    });
    console.time("SET MILLION");
    // for (let index = 0; index < 1000000; index++) {
    //   try {
    //     let num = index;
    //     let key = `akuma_${index}`;
    //     console.log(num);
    //     console.log(key);
    //     kron.set({ key, expire: `${num} seconds`, data: index });
    //   } catch (error) {
    //     console.log(error);
    //   }
    // }
    console.timeEnd("SET MILLION");

    console.log("connected");
  } catch (error) {
    console.log({ error });
  }
}
main();
