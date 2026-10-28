import { describe, expect, it } from "vitest";
import {
  activityWords,
  addDays,
  ago,
  bpField,
  changePercent,
  dayLabel,
  dotColour,
  isDay,
  lapses,
  mondayOf,
  nairobiNow,
  parsePercentBP,
  parseShillings,
  partOfDay,
  signedKes,
  trend,
  when,
} from "./desk";
import { channelBars, statusBar } from "./deskCharts";

const now = "2026-10-28T14:05";

describe("how the desk writes times", () => {
  it("says today, yesterday and tomorrow, then the day, then the year", () => {
    expect(when("2026-10-28T09:30", now)).toBe("Today 09:30");
    expect(when("2026-10-27T18:00", now)).toBe("Yesterday 18:00");
    expect(when("2026-10-29T10:00", now)).toBe("Tomorrow 10:00");
    expect(when("2026-10-21T10:00", now)).toBe("Wed 21 Oct 10:00");
    expect(when("2025-12-02T11:15", now)).toBe("2 Dec 2025");
    expect(when("2027-01-01T08:00", "2026-12-31T23:59")).toBe("Tomorrow 08:00");
  });

  it("says how long ago, as Oriel did", () => {
    expect(ago("2026-10-28T14:05", now)).toBe("a minute");
    expect(ago("2026-10-28T13:50", now)).toBe("15 minutes");
    expect(ago("2026-10-28T09:05", now)).toBe("5 hours");
    expect(ago("2026-10-26T15:00", now)).toBe("47 hours");
    expect(ago("2026-10-20T14:05", now)).toBe("8 days");
  });

  it("names days and weeks in Nairobi, from the strings alone", () => {
    expect(dayLabel("2026-10-21")).toBe("Wed 21 Oct");
    expect(dayLabel("")).toBe("");
    expect(mondayOf("2026-10-28")).toBe("2026-10-26");
    expect(mondayOf("2026-11-01")).toBe("2026-10-26");
    expect(addDays("2026-12-28", 7)).toBe("2027-01-04");
    expect(isDay("2026-02-29")).toBe(false);
    expect(isDay("2028-02-29")).toBe(true);
    expect(nairobiNow(Date.UTC(2026, 9, 28, 11, 5))).toBe(now);
    expect(partOfDay(now)).toBe("afternoon");
    expect(partOfDay("2026-10-28T08:00")).toBe("morning");
    expect(partOfDay("2026-10-28T17:00")).toBe("evening");
  });

  it("says when a hold lapses", () => {
    expect(lapses(5 * 3600 + 100)).toBe("lapses in 5 hours");
    expect(lapses(20 * 60)).toBe("lapses in 20 minutes");
    expect(lapses(30)).toBe("lapses in a minute");
    expect(lapses(0)).toBe("lapsing now");
  });

  it("compares a figure with the one before", () => {
    expect(trend(5, 2, "week")).toBe("3 more than the week before");
    expect(trend(1, 4, "month")).toBe("3 fewer than the month before");
    expect(trend(2, 2, "week")).toBe("the same as the week before");
  });
});

describe("what the desk reads from its forms", () => {
  it("reads shillings typed with or without commas", () => {
    expect(parseShillings("18,480,000")).toBe(18_480_000);
    expect(parseShillings(" KES 18480000 ")).toBe(18_480_000);
    expect(parseShillings("18.4M")).toBeNull();
    expect(parseShillings("")).toBeNull();
  });

  it("reads a percentage into basis points, and writes it back", () => {
    expect(parsePercentBP("0.8")).toBe(80);
    expect(parsePercentBP("1")).toBe(100);
    expect(parsePercentBP("2.25%")).toBe(225);
    expect(parsePercentBP("1.255")).toBeNull();
    expect(parsePercentBP("-1")).toBeNull();
    expect(bpField(80)).toBe("0.8");
    expect(bpField(400)).toBe("4");
  });

  it("writes a change in money with its sign", () => {
    expect(signedKes(120_000)).toBe("+KES 120,000");
    expect(signedKes(-40_000)).toBe("−KES 40,000");
    expect(signedKes(0)).toBe("no change");
    expect(changePercent(1_000_000, 1_024_000)).toBe("+2.4%");
    expect(changePercent(0, 5)).toBe("");
  });

  it("splits a line of the log around its subject", () => {
    expect(activityWords("moved {} to Viewing")).toEqual({ before: "moved ", after: " to Viewing" });
    expect(activityWords("took an enquiry from")).toEqual({ before: "took an enquiry from ", after: "" });
  });

  it("only colours a dot with a plain colour", () => {
    expect(dotColour("#1f3bd1")).toBe("#1f3bd1");
    expect(dotColour("red;background:url(x)")).toBe("#81878c");
  });
});

describe("the desk's charts", () => {
  it("divides every home by status, in the drawings' order", () => {
    const bar = statusBar({ available: 30, sold: 40, reserved: 6, held: 3, withheld: 2 }, 81);
    expect(bar.segments.map((s) => s.status)).toEqual(["sold", "reserved", "held", "available", "withheld"]);
    expect(bar.summary).toBe("81 homes: 40 sold, 6 reserved, 3 on hold, 30 available, 2 not yet released");
  });

  it("draws the channels on one scale, with nothing drawn for none", () => {
    const chart = channelBars([
      { channel: "Search", leads: 20, viewed: 10, reserved: 0 },
      { channel: "Direct", leads: 5, viewed: 1, reserved: 1 },
    ]);
    const search = chart.groups[0]!.bars;
    expect(search[1]!.w).toBeCloseTo(search[0]!.w / 2);
    expect(search[2]!.w).toBe(0);
    expect(chart.summary).toContain("Search: 20 leads, 10 viewed, 0 reserved");
  });
});
