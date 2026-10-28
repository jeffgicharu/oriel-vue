/// <reference types="vite/client" />

declare global {
  /** The commit id of the release, written in when it is built. */
  const __RELEASE__: string;
}

// Templates are checked strictly, so the data attributes the styles and the
// end-to-end tests use are declared.
declare module "vue" {
  interface HTMLAttributes {
    [key: `data-${string}`]: string | number | boolean | undefined;
  }
  interface SVGAttributes {
    [key: `data-${string}`]: string | number | boolean | undefined;
  }
}

export {};
