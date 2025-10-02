// Minimal model declarations matching the Go bindings used in the frontend.

export namespace app {
    export class ServerInfo {
        host: string;
        port: number;
        secret: string;

        constructor(source: any = {}) {
            this.host = source.host ?? "";
            this.port = source.port ?? 0;
            this.secret = source.secret ?? "";
        }
    }

    export class Environment {
        label: string;

        constructor(source: any = {}) {
            this.label = source.label ?? "";
        }
    }
}
