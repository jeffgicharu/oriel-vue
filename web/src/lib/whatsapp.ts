// WhatsApp enquiries. The link opens WhatsApp with a message already written
// that names the home, so the sales team knows at once what it is about.
//
// A real business puts its own number here. This site has none of its own,
// so the link carries no number and WhatsApp asks whom to send it to; no
// real person's number is ever used.

import type { Home } from "@/api/client";
import { kes } from "./format";

export const WHATSAPP_NUMBER = "";

export function whatsappMessage(siteUrl: string, home?: Pick<Home, "id" | "title" | "price"> | null): string {
  if (!home) return `Hello Oriel, I would like to know more about the homes at Oriel, Riverside. ${siteUrl}/residences`;
  const price = home.price !== null ? `, ${kes(home.price)}` : "";
  return `Hello Oriel, I would like to know more about home ${home.id} (${home.title}${price}): ${siteUrl}/residences/${home.id}`;
}

export function whatsappHref(message: string): string {
  return `https://wa.me/${WHATSAPP_NUMBER}?text=${encodeURIComponent(message)}`;
}
