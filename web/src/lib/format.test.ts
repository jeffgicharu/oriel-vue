import { describe, expect, it } from "vitest";
import { drawn, kes, kesShort, longDate, longDateTime, monthYear, ordinal, percent, thousands } from "./format";

describe("money", () => {
  it("writes shillings with commas", () => {
    expect(thousands(18480000)).toBe("18,480,000");
    // A no-break space keeps "KES" on the line with its number.
    expect(kes(950)).toBe("KES\u00a0950");
    expect(kes(-1200)).toBe("KES\u00a0-1,200");
  });

  it("writes prices for headings to two exact decimals", () => {
    expect(kesShort(17_600_000)).toBe("KES\u00a017.60M");
    expect(kesShort(11_400_000)).toBe("KES\u00a011.40M");
    expect(kesShort(1_250_000_000)).toBe("KES\u00a01.25B");
    expect(kesShort(65_000)).toBe("KES\u00a065K");
  });

  it("writes basis points as a percentage", () => {
    expect(percent(1350)).toBe("13.5%");
    expect(percent(700)).toBe("7%");
    expect(percent(571)).toBe("5.71%");
  });
});

describe("dates, the same on the server and in any browser", () => {
  it("reads a Nairobi day without the runtime's time zone", () => {
    expect(longDate("2028-12-15")).toBe("Friday 15 December 2028");
    expect(monthYear("2028-12-15")).toBe("December 2028");
    expect(longDateTime("2026-10-29T10:30")).toBe("Thursday 29 October, 10:30");
    expect(drawn("2026-10-08")).toBe("08.10.2026");
  });

  it("writes levels as ordinals", () => {
    expect(ordinal(1)).toBe("1st");
    expect(ordinal(12)).toBe("12th");
    expect(ordinal(22)).toBe("22nd");
  });
});
