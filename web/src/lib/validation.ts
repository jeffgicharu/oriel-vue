// The checks the forms make before sending, the same as the API makes. The
// browser's checks spare a round trip; the API's are the ones that count,
// and its answers land beside the same fields.

export type FieldErrors = Record<string, string>;

const phoneDigits = /^\+?[0-9]{9,15}$/;

/** Writes a Kenyan number as +254 7XX XXX XXX, and leaves international numbers as they are. */
export function normalisePhone(raw: string): string | null {
  let s = raw.replace(/[\s\-().]/g, "");
  if (!phoneDigits.test(s)) return null;
  if (s.startsWith("0") && s.length === 10) s = "+254" + s.slice(1);
  else if (s.startsWith("254") && s.length === 12) s = "+" + s;
  else if ((s.startsWith("7") || s.startsWith("1")) && s.length === 9) s = "+254" + s;
  if (s.startsWith("+254") && s.length === 13) return `${s.slice(0, 4)} ${s.slice(4, 7)} ${s.slice(7, 10)} ${s.slice(10)}`;
  return s.startsWith("+") ? s : null;
}

export interface ContactValues {
  name: string;
  email: string;
  phone: string;
}

export function contactErrors(v: ContactValues): FieldErrors {
  const errs: FieldErrors = {};
  const name = v.name.trim().split(/\s+/).filter(Boolean).join(" ");
  const n = [...name].length;
  if (n === 0) errs.name = "Please tell us your name.";
  else if (n < 2) errs.name = "Please give your full name.";
  else if (n > 80) errs.name = "That name is too long.";

  const email = v.email.trim();
  const at = email.lastIndexOf("@");
  if (email === "") errs.email = "Please give an email address.";
  else if (at < 1 || !email.slice(at + 1).includes(".") || /\s/.test(email) || email.length > 120) errs.email = "That email address doesn’t look right.";

  const phone = v.phone.trim();
  if (phone === "") errs.phone = "Please give a phone number.";
  else if (!normalisePhone(phone)) errs.phone = "Please give a phone number, such as 0712 345 678 or +254 712 345 678.";
  return errs;
}

export function textErrors(field: string, value: string, opts: { required?: string; max: number; tooLong: string }): FieldErrors {
  const v = value.trim();
  if (opts.required && v === "") return { [field]: opts.required };
  if ([...v].length > opts.max) return { [field]: opts.tooLong };
  return {};
}
