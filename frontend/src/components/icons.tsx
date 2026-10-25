// Small stroke icons (currentColor, 24 viewBox) shared across the UI.
function Base({ children, label }: { children: React.ReactNode; label?: string }) {
  return (
    <svg
      width="17"
      height="17"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden={label ? undefined : true}
      role={label ? "img" : undefined}
      aria-label={label}
      style={{ flex: "none", verticalAlign: "-3px" }}
    >
      {children}
    </svg>
  );
}

export function IconUser() {
  return (
    <Base>
      <circle cx="12" cy="8" r="3.6" />
      <path d="M4.5 20c1.4-3.4 4.2-5 7.5-5s6.1 1.6 7.5 5" />
    </Base>
  );
}

export function IconHeart({ filled }: { filled?: boolean }) {
  return (
    <svg
      width="17"
      height="17"
      viewBox="0 0 24 24"
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      style={{ flex: "none", verticalAlign: "-3px" }}
    >
      <path d="M12 20.5C6 16.5 3 13.3 3 9.6 3 7 5 5 7.4 5c1.8 0 3.6 1 4.6 2.6C13 6 14.8 5 16.6 5 19 5 21 7 21 9.6c0 3.7-3 6.9-9 10.9Z" />
    </svg>
  );
}

export function IconList() {
  return (
    <Base>
      <path d="M8.5 6h11M8.5 12h11M8.5 18h11" />
      <circle cx="4.5" cy="6" r="1.1" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="12" r="1.1" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="18" r="1.1" fill="currentColor" stroke="none" />
    </Base>
  );
}

export function IconShield() {
  return (
    <Base>
      <path d="M12 3.5 5 6v5.2c0 4.4 3 7.6 7 9.3 4-1.7 7-4.9 7-9.3V6l-7-2.5Z" />
      <path d="m9 11.6 2.1 2.1 4-4.2" />
    </Base>
  );
}

export function IconLink() {
  return (
    <Base>
      <path d="M10.5 13.5a4.5 4.5 0 0 0 6.4 0l3-3a4.5 4.5 0 1 0-6.4-6.4l-1.5 1.5" />
      <path d="M13.5 10.5a4.5 4.5 0 0 0-6.4 0l-3 3a4.5 4.5 0 1 0 6.4 6.4l1.5-1.5" />
    </Base>
  );
}

export function IconCheck() {
  return (
    <Base>
      <path d="m5 12.5 4.5 4.5L19 7.5" />
    </Base>
  );
}

export function IconUpload() {
  return (
    <Base>
      <path d="M12 16V4.5M6.8 9 12 3.8 17.2 9" />
      <path d="M4.5 16.5v2A1.5 1.5 0 0 0 6 20h12a1.5 1.5 0 0 0 1.5-1.5v-2" />
    </Base>
  );
}

export function IconCamera() {
  return (
    <Base>
      <path d="M4 8.5A1.5 1.5 0 0 1 5.5 7h1L8 4.8A1.5 1.5 0 0 1 9.3 4h5.4A1.5 1.5 0 0 1 16 4.8L17.5 7h1A1.5 1.5 0 0 1 20 8.5v9A1.5 1.5 0 0 1 18.5 19h-13A1.5 1.5 0 0 1 4 17.5v-9Z" />
      <circle cx="12" cy="12.5" r="3.2" />
    </Base>
  );
}
