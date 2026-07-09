import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    server: {
      deps: {
        inline: [
          '@mui/material',
          '@mui/icons-material',
          '@mui/system',
          '@emotion/react',
          '@emotion/styled',
        ],
      },
    },
  },
})
