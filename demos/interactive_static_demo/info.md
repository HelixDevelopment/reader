Declared in `package.json`: Tailwind CSS `^3.4.19`, Vite `^7.2.4`, React `^19.2.0`,
TypeScript `~5.9.3`. No Node.js version is pinned (no `engines`, `.nvmrc`, or
`.node-version`); the toolchain actually used is recorded in `evidence/` per run.

This directory is the React/Vite responsive prototype. It is intentionally
disconnected from the server and uses sample data.

Tailwind CSS is configured with the shadcn theme (`tailwind.config.js`,
`components.json`).

Components (40+):
  accordion, alert-dialog, alert, aspect-ratio, avatar, badge, breadcrumb,
  button-group, button, calendar, card, carousel, chart, checkbox, collapsible,
  command, context-menu, dialog, drawer, dropdown-menu, empty, field, form,
  hover-card, input-group, input-otp, input, item, kbd, label, menubar,
  navigation-menu, pagination, popover, progress, radio-group, resizable,
  scroll-area, select, separator, sheet, sidebar, skeleton, slider, sonner,
  spinner, switch, table, tabs, textarea, toggle-group, toggle, tooltip

Usage:
  import { Button } from '@/components/ui/button'
  import { Card, CardHeader, CardTitle } from '@/components/ui/card'

Structure:
  src/components/      UI components (src/components/ui/ is the shadcn library)
  src/hooks/           Custom hooks
  src/lib/             Utilities
  src/App.css          Styles specific to the Webapp
  src/App.tsx          Root React component
  src/index.css        Global styles
  src/main.tsx         Entry point for rendering the Webapp
  index.html           Entry point for the Webapp
  tailwind.config.js   Configures Tailwind's theme, plugins, etc.
  vite.config.ts       Main build and dev server settings for Vite
  postcss.config.js    Config file for CSS post-processing tools