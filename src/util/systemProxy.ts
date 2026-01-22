import {GetUsername} from "@/runtime";

type SystemProxySettings = {
  bindAddress: string;
  port: number;
};

export async function updateSystemProxy(
  api: any,
  settings: SystemProxySettings,
  enable: boolean,
) {
  const username = GetUsername();
  console.log('[SystemProxy] Current username:', username);

  if (enable) {
    console.log('[SystemProxy] Enabling proxy for', username, 'at', settings.bindAddress + ':' + settings.port);
    return api.enableProxy({
      bindAddress: settings.bindAddress,
      port: settings.port,
      username: username,
    });
  }

  console.log('[SystemProxy] Disabling proxy for', username);
  return api.disableProxy({
    username: username,
  });
}
