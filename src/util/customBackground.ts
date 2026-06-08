const USER_IMAGE_PREFIX = '/user-images/';

const parseHttpOrigin = (value: string | null) => {
  if (!value || value === 'null' || value === 'undefined') {
    return null;
  }

  try {
    const parsed = new URL(value);
    if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
      return `${parsed.protocol}//${parsed.host}`;
    }
  } catch (error) {
    try {
      const parsed = new URL(`http://${value}`);
      if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
        return `${parsed.protocol}//${parsed.host}`;
      }
    } catch {
      return null;
    }
  }

  return null;
};

export const getRendererOrigin = () => {
  if (typeof window === 'undefined') {
    return null;
  }

  const params = new URLSearchParams(window.location.search);
  const fromParam = parseHttpOrigin(params.get('frontendOrigin'));
  const fromLocation = parseHttpOrigin(window.location.origin);

  return fromParam ?? fromLocation;
};

export const buildRendererUrl = (path: string, rendererOrigin: string | null) => {
  if (!rendererOrigin) {
    return path.startsWith('/') ? path : `/${path}`;
  }

  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${rendererOrigin}${normalizedPath}`;
};

export const extractUrlFromCssValue = (value: string | null) => {
  if (!value) {
    return null;
  }
  const match = value.match(/^url\(['"]?(.*?)['"]?\)$/);
  if (!match) {
    return null;
  }
  return match[1];
};

export const getRelativeUserImagePath = (value: string | null, rendererOrigin: string | null) => {
  const url = extractUrlFromCssValue(value);
  if (!url) {
    return null;
  }

  if (url.startsWith(USER_IMAGE_PREFIX)) {
    return url;
  }

  if (rendererOrigin && url.startsWith(rendererOrigin)) {
    const candidate = url.slice(rendererOrigin.length);
    if (candidate.startsWith(USER_IMAGE_PREFIX)) {
      return candidate;
    }
  }

  try {
    const parsed = new URL(url);
    if (parsed.pathname.startsWith(USER_IMAGE_PREFIX)) {
      return parsed.pathname;
    }
  } catch {
    // ignore invalid urls
  }

  return null;
};

export const ensureRelativeStorageValue = (value: string | null, rendererOrigin: string | null) => {
  if (!value) {
    return null;
  }

  const relative = getRelativeUserImagePath(value, rendererOrigin);
  if (relative) {
    return `url('${relative}')`;
  }

  return value;
};

export const makeCssAbsoluteForUse = (value: string, rendererOrigin: string | null) => {
  const relative = getRelativeUserImagePath(value, rendererOrigin);
  if (relative && rendererOrigin) {
    return `url('${buildRendererUrl(relative, rendererOrigin)}')`;
  }
  return value;
};

export const normalizeResponsePath = (value: string) => {
  if (value.startsWith(USER_IMAGE_PREFIX)) {
    return value;
  }

  try {
    const parsed = new URL(value);
    if (parsed.pathname.startsWith(USER_IMAGE_PREFIX)) {
      return parsed.pathname;
    }
  } catch {
    // ignore
  }

  throw new Error('Invalid custom background path received from server');
};

export const createStorageValue = (relativePath: string) => `url('${relativePath}')`;

export const normalizeCustomBackground = (value: string | null, rendererOrigin: string | null) => {
  if (!value) {
    return null;
  }

  const relativePath = getRelativeUserImagePath(value, rendererOrigin);
  if (!relativePath) {
    return {storageValue: value, cssValue: value};
  }

  const storageValue = createStorageValue(relativePath);
  const cssValue = makeCssAbsoluteForUse(storageValue, rendererOrigin);

  return {storageValue, cssValue};
};
