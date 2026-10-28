import { beforeEach, describe, expect, it } from "vitest";
import { currentSource, recordArrival, touchFrom } from "./leadSource";

const site = "https://oriel-vue.example.com";

describe("where a visitor came from", () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it("reads the campaign from the address and the sending site from the referrer", () => {
    const t = touchFrom(new URL(site + "/residences/0704?utm_source=instagram&utm_medium=paid-social&utm_campaign=riverside-launch"), "https://l.instagram.com/", new Date("2026-10-28T10:00:00Z"));
    expect(t).toEqual({
      source: "instagram",
      medium: "paid-social",
      campaign: "riverside-launch",
      referrer: "https://l.instagram.com/",
      landing: "/residences/0704",
      at: "2026-10-28T10:00:00.000Z",
    });
  });

  it("does not count the site's own pages as a referrer", () => {
    expect(touchFrom(new URL(site + "/buying"), site + "/residences", new Date()).referrer).toBe("");
  });

  it("keeps the first arrival and moves the last one on with a new campaign", () => {
    recordArrival(new URL(site + "/?utm_source=google&utm_medium=cpc"), "", new Date("2026-10-01T09:00:00Z"));
    recordArrival(new URL(site + "/residences"), "", new Date("2026-10-01T09:05:00Z"));
    expect(currentSource()?.first.source).toBe("google");
    expect(currentSource()?.last.source).toBe("google");

    recordArrival(new URL(site + "/residences/1601?utm_source=newsletter&utm_medium=email"), "", new Date("2026-10-20T18:00:00Z"));
    expect(currentSource()?.first.source).toBe("google");
    expect(currentSource()?.last.source).toBe("newsletter");
  });

  it("starts a new first arrival after 90 days", () => {
    recordArrival(new URL(site + "/?utm_source=google"), "", new Date("2026-01-01T09:00:00Z"));
    sessionStorage.clear();
    recordArrival(new URL(site + "/"), "https://www.bing.com/", new Date("2026-06-01T09:00:00Z"));
    expect(currentSource()?.first.referrer).toBe("https://www.bing.com/");
  });
});
