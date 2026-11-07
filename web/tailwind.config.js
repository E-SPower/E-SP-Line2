/** @type {import('tailwindcss').Config} */
const withOpacity = (variable) => `hsl(var(${variable}) / <alpha-value>)`;

module.exports = {
  darkMode: ["class"],
  content: [
    './index.html',
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  prefix: "",
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      // ---- colour tokens (mirror of the CSS variables in src/index.css) --
      colors: {
        surface: {
          DEFAULT: withOpacity("--surface"),
          raised: withOpacity("--surface-raised"),
          sunken: withOpacity("--surface-sunken"),
          inset: withOpacity("--surface-inset"),
          overlay: withOpacity("--surface-overlay"),
          hover: withOpacity("--surface-hover"),
          active: withOpacity("--surface-active"),
        },
        text: {
          primary: withOpacity("--text-primary"),
          secondary: withOpacity("--text-secondary"),
          tertiary: withOpacity("--text-tertiary"),
          disabled: withOpacity("--text-disabled"),
          inverse: withOpacity("--text-inverse"),
        },
        border: withOpacity("--border"),
        "border-strong": withOpacity("--border-strong"),
        "border-subtle": withOpacity("--border-subtle"),
        ring: withOpacity("--ring"),
        "ring-offset": withOpacity("--ring-offset"),

        primary: {
          DEFAULT: withOpacity("--primary"),
          foreground: withOpacity("--primary-foreground"),
          hover: withOpacity("--primary-hover"),
          active: withOpacity("--primary-active"),
          subtle: withOpacity("--primary-subtle"),
          "subtle-foreground": withOpacity("--primary-subtle-foreground"),
        },
        accent: {
          DEFAULT: withOpacity("--accent"),
          foreground: withOpacity("--accent-foreground"),
          subtle: withOpacity("--accent-subtle"),
          "subtle-foreground": withOpacity("--accent-subtle-foreground"),
        },
        success: {
          DEFAULT: withOpacity("--success-solid"),
          bg: withOpacity("--success-bg"),
          fg: withOpacity("--success-fg"),
          border: withOpacity("--success-border"),
        },
        warning: {
          DEFAULT: withOpacity("--warning-solid"),
          bg: withOpacity("--warning-bg"),
          fg: withOpacity("--warning-fg"),
          border: withOpacity("--warning-border"),
        },
        danger: {
          DEFAULT: withOpacity("--danger-solid"),
          bg: withOpacity("--danger-bg"),
          fg: withOpacity("--danger-fg"),
          border: withOpacity("--danger-border"),
        },
        info: {
          DEFAULT: withOpacity("--info-solid"),
          bg: withOpacity("--info-bg"),
          fg: withOpacity("--info-fg"),
          border: withOpacity("--info-border"),
        },

        // ---- legacy aliases used by older markup ------------------------
        background: withOpacity("--background"),
        foreground: withOpacity("--foreground"),
        card: {
          DEFAULT: withOpacity("--card"),
          foreground: withOpacity("--card-foreground"),
        },
        popover: {
          DEFAULT: withOpacity("--popover"),
          foreground: withOpacity("--popover-foreground"),
        },
        secondary: {
          DEFAULT: withOpacity("--secondary"),
          foreground: withOpacity("--secondary-foreground"),
        },
        muted: {
          DEFAULT: withOpacity("--muted"),
          foreground: withOpacity("--muted-foreground"),
        },
        destructive: {
          DEFAULT: withOpacity("--destructive"),
          foreground: withOpacity("--destructive-foreground"),
        },
        input: withOpacity("--input"),
      },

      // ---- radius ladder: everything derives from --radius (shadcn) --------
      borderRadius: {
        DEFAULT: "var(--radius-md)",
        xs: "var(--radius-xs)",
        sm: "var(--radius-sm)",
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
        xl: "var(--radius-xl)",
        "2xl": "var(--radius-2xl)",
        full: "var(--radius-full)",
      },

      // ---- shadow ladder --------------------------------------------------
      boxShadow: {
        xs: "var(--shadow-xs)",
        sm: "var(--shadow-sm)",
        md: "var(--shadow-md)",
        lg: "var(--shadow-lg)",
        xl: "var(--shadow-xl)",
        overlay: "var(--shadow-overlay)",
        none: "none",
      },

      // ---- type scale -----------------------------------------------------
      fontSize: {
        "2xs": ["0.6875rem", { lineHeight: "1rem", letterSpacing: "0.04em" }],
        xs: ["0.75rem", { lineHeight: "1.1rem" }],
        sm: ["0.8125rem", { lineHeight: "1.25rem" }],
        base: ["0.875rem", { lineHeight: "1.5rem" }],
        md: ["0.9375rem", { lineHeight: "1.55rem" }],
        lg: ["1.0625rem", { lineHeight: "1.65rem" }],
        xl: ["1.25rem", { lineHeight: "1.75rem" }],
        "2xl": ["1.5rem", { lineHeight: "1.9rem" }],
        "3xl": ["1.875rem", { lineHeight: "2.2rem", letterSpacing: "-0.015em" }],
        "4xl": ["2.25rem", { lineHeight: "2.5rem", letterSpacing: "-0.02em" }],
      },

      // ---- rhythm ---------------------------------------------------------
      spacing: {
        sidebar: "var(--sidebar-width)",
        "sidebar-collapsed": "var(--sidebar-width-collapsed)",
        topbar: "var(--topbar-height)",
        gutter: "var(--gutter)",
        row: "var(--density-row)",
      },
      maxWidth: {
        content: "var(--content-max)",
      },

      // ---- motion ---------------------------------------------------------
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        "fade-in": {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        "slide-in-left": {
          from: { transform: "translateX(-100%)" },
          to: { transform: "translateX(0)" },
        },
        shimmer: {
          "100%": { transform: "translateX(100%)" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "fade-in": "fade-in 0.15s ease-out",
        "slide-in-left": "slide-in-left 0.2s ease-out",
        shimmer: "shimmer 1.6s infinite",
      },
      transitionTimingFunction: {
        standard: "cubic-bezier(0.2, 0, 0, 1)",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}
