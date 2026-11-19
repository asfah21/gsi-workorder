import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GSI Work Order",
  description: "Sistem work order dan maintenance PT GSI",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id">
      <body>{children}</body>
    </html>
  );
}
