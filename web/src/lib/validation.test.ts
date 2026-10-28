import { describe, expect, it } from "vitest";
import { contactErrors, normalisePhone, textErrors } from "./validation";

describe("the forms' own checks, the same as the API's", () => {
  it("writes Kenyan numbers one way", () => {
    expect(normalisePhone("0712 345 678")).toBe("+254 712 345 678");
    expect(normalisePhone("254712345678")).toBe("+254 712 345 678");
    expect(normalisePhone("712345678")).toBe("+254 712 345 678");
    expect(normalisePhone("+44 20 7946 0958")).toBe("+442079460958");
    expect(normalisePhone("call me")).toBeNull();
  });

  it("names every field that needs another look", () => {
    expect(contactErrors({ name: " ", email: "amani@", phone: "12" })).toEqual({
      name: "Please tell us your name.",
      email: "That email address doesn’t look right.",
      phone: "Please give a phone number, such as 0712 345 678 or +254 712 345 678.",
    });
    expect(contactErrors({ name: "Amani Wanjiru", email: "amani@example.com", phone: "0712345678" })).toEqual({});
  });

  it("checks free text for being there and its length", () => {
    expect(textErrors("message", "", { required: "Please write your question.", max: 10, tooLong: "Too long." })).toEqual({ message: "Please write your question." });
    expect(textErrors("notes", "x".repeat(11), { max: 10, tooLong: "Too long." })).toEqual({ notes: "Too long." });
  });
});
