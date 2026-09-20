const baseUrl = new URL(import.meta.env.BASE_URL, document.baseURI);

export const withBasePath = (path: string) => {
  if (/^[a-z][a-z\d+.-]*:/i.test(path) || path.startsWith("//")) {
    return path;
  }

  const url = new URL(path.replace(/^\/+/, ""), baseUrl);
  return `${url.pathname}${url.search}${url.hash}`;
};
