// Stroke icons used across the app. Decorative by default; give the parent
// control an aria-label when the icon is its only content.

const paths = {
  home: <path d="M3 11l9-7 9 7v9H3z" />,
  ticket: <path d="M3 8a2 2 0 0 0 0 4v4h18v-4a2 2 0 0 1 0-4V4H3z" />,
  user: (
    <>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4 4-6 8-6s8 2 8 6" />
    </>
  ),
  plus: <path d="M12 5v14M5 12h14" />,
  search: (
    <>
      <circle cx="11" cy="11" r="7" />
      <path d="M20 20l-4-4" />
    </>
  ),
  sliders: <path d="M4 6h16M7 12h10M10 18h4" />,
  chat: <path d="M4 20l1.4-4.2A8 8 0 1 1 8.2 18.6L4 20z" />,
  back: <path d="M15 5l-7 7 7 7" />,
  share: <path d="M12 3v12M7 8l5-5 5 5M5 14v6h14v-6" />,
  up: <path d="M12 19V5M5 12l7-7 7 7" />,
  check: <path d="M5 12l5 5L20 7" />,
}

export type IconName = keyof typeof paths

export default function Icon({ name, size = 20, stroke = 2.2 }: { name: IconName; size?: number; stroke?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {paths[name]}
    </svg>
  )
}
