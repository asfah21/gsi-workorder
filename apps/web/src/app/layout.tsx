import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Script from "next/script";

import { TooltipProvider } from "@/components/ui/tooltip";

import "./globals.css";

const geistSans = Geist({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-geist",
});

const geistMono = Geist_Mono({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-geist-mono",
});

export const metadata: Metadata = {
  title: "GSI Work Order",
  description: "Sistem work order dan maintenance PT GSI",
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
      className={`${geistSans.variable} ${geistMono.variable}`}
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
