import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/setupTests.ts'],
      globals: true,
      // MUI's ESM build imports react-transition-group via a bare directory
      // specifier (e.g. "react-transition-group/TransitionGroupContext")
      // that only resolves through Vite's bundler-style resolution, not
      // Node's native ESM resolver. Forcing these packages to be processed
      // by Vite (instead of externalized) avoids a "Directory import is not
      // supported" error under vitest's jsdom pool.
      server: {
        deps: {
          inline: [/@mui\//, /react-transition-group/],
        },
      },
    },
  })
)
