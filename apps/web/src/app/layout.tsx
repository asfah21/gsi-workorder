import type { Metadata } from "next";
import localFont from "next/font/local";
import Script from "next/script";

import { TooltipProvider } from "@/components/ui/tooltip";

import "./globals.css";

const Geist = localFont({
  src: "./fonts/Geist-Variable.woff2",
  variable: "--font-geist",
  weight: "100 900",
  display: "swap",
});

const GeistMono = localFont({
  src: "./fonts/GeistMono-Variable.woff2",
  variable: "--font-geist-mono",
  weight: "100 900",
  display: "swap",
  adjustFontFallback: false,
  fallback: [
    "ui-monospace",
    "SFMono-Regular",
    "Menlo",
    "Monaco",
    "Consolas",
    "monospace",
  ],
});

export const metadata: Metadata = {
  title: "PAMS | Plant and Maintenance System",
  description:
    "Sistem operasional Plant and Maintenance GSI untuk work order, PM, armada, inventory, dan laporan.",
  other: {
    "color-scheme": "light dark",
  },
};

const themeInitScript = `(function(){try{var key="gsi-theme";var stored=localStorage.getItem(key);var theme=stored==="dark"||stored==="light"?stored:(window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light");var root=document.documentElement;root.classList.toggle("dark",theme==="dark");root.style.colorScheme=theme}catch(e){}})();`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="id"
      suppressHydrationWarning
      data-card-shadow="false"
      data-sidebar-type="true"
      className={`${Geist.variable} ${GeistMono.variable}`}
    >
      <body className="font-sans">
        <Script id="gsi-theme-init" strategy="beforeInteractive">
          {themeInitScript}
        </Script>
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  );
}
