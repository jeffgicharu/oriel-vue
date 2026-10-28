// Structured data for search engines and link previews, as schema.org
// describes it. Pages set it through usePageMeta, so it is in the HTML the
// server sends.

import type { Home } from "@/api/client";
import { monthYear } from "./format";
import { showsPrice } from "./homes";
import iconUrl from "@/assets/img/icon-512.png";

type LD = Record<string, unknown>;

export function organizationLd(site: string): LD {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": `${site}/#organization`,
    name: "Oriel Residences Limited",
    brand: "Oriel",
    url: `${site}/`,
    logo: site + iconUrl,
    address: { "@type": "PostalAddress", addressLocality: "Nairobi", addressRegion: "Nairobi", addressCountry: "KE" },
  };
}

export function websiteLd(site: string): LD {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    "@id": `${site}/#website`,
    name: "Oriel, Riverside",
    url: `${site}/`,
    inLanguage: "en-KE",
    publisher: { "@id": `${site}/#organization` },
  };
}

export function complexLd(site: string, completion: string): LD {
  return {
    "@context": "https://schema.org",
    "@type": "ApartmentComplex",
    "@id": `${site}/#oriel`,
    name: "Oriel",
    description: `A seventeen-storey building of 86 homes, two of them penthouses, in Riverside, Nairobi, completing in ${monthYear(completion)}.`,
    url: `${site}/`,
    numberOfAccommodationUnits: 86,
    petsAllowed: false,
    address: { "@type": "PostalAddress", addressLocality: "Riverside, Nairobi", addressCountry: "KE" },
    amenityFeature: [
      { "@type": "LocationFeatureSpecification", name: "Rooftop pool", value: true },
      { "@type": "LocationFeatureSpecification", name: "Gym", value: true },
      { "@type": "LocationFeatureSpecification", name: "Residents' lounge and work room", value: true },
      { "@type": "LocationFeatureSpecification", name: "Backup power and water", value: true },
    ],
  };
}

export function apartmentLd(site: string, h: Home, summary: string): LD {
  const url = `${site}/residences/${h.id}`;
  const ld: LD = {
    "@context": "https://schema.org",
    "@type": "Apartment",
    name: `Home ${h.id} at Oriel`,
    description: summary,
    url,
    numberOfRooms: h.bedrooms + 1,
    numberOfBedrooms: h.bedrooms,
    numberOfBathroomsTotal: h.bathrooms,
    floorLevel: String(h.level),
    floorSize: { "@type": "QuantitativeValue", value: h.internal, unitCode: "MTK" },
    containedInPlace: { "@id": `${site}/#oriel` },
  };
  if (showsPrice(h) && h.price !== null) {
    ld.offers = {
      "@type": "Offer",
      price: h.price,
      priceCurrency: "KES",
      availability: h.status === "held" ? "https://schema.org/LimitedAvailability" : "https://schema.org/InStock",
      url,
    };
  }
  return ld;
}

export function breadcrumbLd(site: string, trail: { name: string; path: string }[]): LD {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: trail.map((t, i) => ({ "@type": "ListItem", position: i + 1, name: t.name, item: site + t.path })),
  };
}
