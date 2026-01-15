type SystemProxySettings = {
  bindAddress: string;
  port: number;
};

export async function updateSystemProxy(
  api: any,
  settings: SystemProxySettings,
  enable: boolean,
) {
  if (enable) {
    return api.enableProxy({
      bindAddress: settings.bindAddress,
      port: settings.port,
    });
  }

  return api.disableProxy();
}
