export type ProxySwitchApi = {
  setProxy: (group: string, payload: {name: string}) => Promise<unknown>;
  closeAllConnection?: () => Promise<unknown> | unknown;
};

export async function changeProxyAndCloseConnections(
  api: ProxySwitchApi,
  groupName: string,
  proxyName: string,
) {
  await api.setProxy(groupName, {name: proxyName});

  if (typeof api.closeAllConnection !== 'function') {
    return;
  }

  try {
    await api.closeAllConnection();
  } catch (error) {
    console.error('Failed to close connections after switching proxy', error);
  }
}
