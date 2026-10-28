// The words of the content pages that are not about one home: what the
// residents share, the specification, the steps of buying, the costs and the
// questions buyers ask.

import type { Schedule } from "@/api/client";
import { monthYear } from "./format";

export interface Amenity {
  code: string;
  name: string;
  where: string;
  text: string;
  image: string;
  alt: string;
}

/** The shared spaces, from the roof down. */
export const amenities: Amenity[] = [
  { code: "R", name: "Rooftop pool and terrace", where: "Roof", text: "A twenty-metre pool with a pale stone edge, a timber deck and a slatted pergola for shade, with the treetops of Riverside on every side.", image: "rooftop-pool", alt: "A long pool on the roof with a timber deck, loungers under a slatted pergola and treetops beyond" },
  { code: "G", name: "Residents’ gym", where: "Level 1", text: "Cardio, free weights and a stretching studio behind a glass wall onto the garden. Open from 5 in the morning to 11 at night.", image: "gym", alt: "A woman on a rowing machine in a gym with bikes and weights, behind a glass wall onto the garden" },
  { code: "L", name: "Lounge and workroom", where: "Level 1", text: "Quiet desks, a long table and a meeting room residents can book, with fibre and a kitchen for coffee.", image: "lounge-workroom", alt: "Residents working at a long oak table under pendant lights, with lounge chairs and a cobalt armchair by the window" },
  { code: "E", name: "Lobby and concierge", where: "Ground", text: "A lobby four and a half metres high, a concierge desk staffed day and night, and a cool room for parcels and deliveries.", image: "lobby", alt: "A tall lobby with board-marked concrete walls, a terrazzo floor and a concierge at an oak desk" },
  { code: "W", name: "Garden and river walk", where: "Ground", text: "A planted garden that runs down to a path along the river, kept wild at the water’s edge.", image: "river-walk", alt: "A runner on a path beside a misty river, under tall trees" },
];

export interface SpecGroup {
  code: string;
  title: string;
  items: [name: string, text: string][];
}

/** What the building is made of and what every home comes with. */
export const specification: SpecGroup[] = [
  { code: "S1", title: "Structure and envelope", items: [
    ["Frame", "Reinforced concrete frame and flat slabs, designed for Nairobi’s seismic zone."],
    ["Façade", "Smooth off-white render, with deep concrete fins shading the east and west faces."],
    ["Bays", "Glazed bays in graphite thermally broken aluminium, under a cobalt powder-coated band."],
    ["Glazing", "Double glazing with a low-emissivity coating throughout; tilt-and-turn windows that open."],
    ["Balconies", "Frameless glass balustrades with a slim graphite rail, and terrazzo paving."],
  ] },
  { code: "S2", title: "Inside every home", items: [
    ["Floors", "Engineered oak in living rooms and bedrooms; terrazzo in kitchens, bathrooms and halls."],
    ["Ceilings", "2.8 metres clear in living rooms, lowered over each bay to frame the window seat."],
    ["Doors", "Full-height doors in oak veneer, with brushed brass handles."],
    ["Feature walls", "The concrete of the core left exposed where it meets the living room, board-marked and sealed."],
    ["Wardrobes", "Built in, in oak veneer, in every bedroom."],
    ["Window seat", "Solid oak, with storage below and a cushion in the bay."],
  ] },
  { code: "S3", title: "Kitchens and bathrooms", items: [
    ["Cabinets", "Handleless, in oak veneer or matt graphite, chosen during fit-out."],
    ["Worktops", "20 mm quartz with a matching upstand."],
    ["Appliances", "Induction hob, oven, extractor, integrated fridge-freezer and dishwasher; a washer-dryer in the laundry."],
    ["Bathrooms", "Terrazzo floors and walls, a wall-hung WC, a walk-in shower behind a glass screen, and a bath in the main bathroom of three-bedroom homes and penthouses."],
    ["Fittings", "Brushed brass mixers and showers, a heated towel rail and a round lit mirror."],
  ] },
  { code: "S4", title: "Services", items: [
    ["Water", "Borehole and mains, with a day’s storage for the whole building. Rainwater is kept for the garden."],
    ["Hot water", "Solar water heating on the roof, with an electric top-up."],
    ["Power", "A backup generator for every home, the lifts, the pumps and the common areas."],
    ["Lifts", "Two passenger lifts and a goods lift, reaching both basement levels and the roof."],
    ["Connectivity", "Fibre to every home from two providers."],
    ["Security", "Gatehouse staffed day and night, access control at the lobby and lifts, cameras in the common areas, and a video entryphone in each home."],
    ["Parking", "Two basement levels: one space for each one- and two-bedroom home, two for three-bedroom homes and penthouses, and charging points for electric cars."],
  ] },
];

export interface BuyingStep {
  title: string;
  when: string;
  text: string;
}

/** The steps from a viewing to the keys; the last is dated by the programme as it stands. */
export function buyingSteps(completion: string): BuyingStep[] {
  return [
    { title: "Choose your home", when: "Any time", text: "Explore the building, shortlist homes and come to the sales suite to see the show apartments and the model." },
    { title: "Request a reservation", when: "Day 1", text: "Choose a payment plan. The home is held for you for 72 hours while the sales team checks it is free and prepares a letter of offer." },
    { title: "Pay the deposit", when: "Within 7 days", text: "Pay 10% by bank transfer to the project’s escrow account. The home is then yours, and it comes off the market." },
    { title: "Sign the sale agreement", when: "Within 30 days", text: "Your advocate reviews the agreement for sale; you sign and pay the next instalment as your plan sets out." },
    { title: "Pay the instalments", when: "Until completion", text: "As your plan sets out: in stages as the building rises, within 90 days if you pay cash, or from your bank at completion. You can follow every stage on the progress page." },
    { title: "Collect your keys", when: completion ? monthYear(completion) : "", text: "We walk through the home with you, fix anything on the snagging list, and hand over the keys and your title." },
  ];
}

/** Costs paid on top of the price. */
export const buyingCosts: [name: string, amount: string, note: string][] = [
  ["Stamp duty", "4% of the price", "Paid to the government when the transfer is registered. It is 4% for homes in a town or city."],
  ["Advocate’s fees", "About 1% to 1.5%, plus VAT", "Set by the Advocates Remuneration Order. You choose your own advocate."],
  ["Service charge deposit", "Three months’ charge", "Held by the management company. The charge is estimated at KES\u00a065 a square metre a month."],
  ["Utility deposits", "About KES\u00a015,000", "For water and electricity meters in your name."],
  ["Mortgage costs", "If you borrow", "Your bank’s arrangement and valuation fees, and stamp duty on the charge, usually 0.1%."],
];

/** The questions buyers ask most. */
export const questions: [q: string, a: string][] = [
  ["Can I buy from outside Kenya?", "Yes. Many of our buyers live abroad. You can view by video call, sign documents with a notary or at a Kenyan embassy, and pay by international bank transfer. Each home is held as a lease, which buyers who are not Kenyan citizens can own."],
  ["What title will I get?", "Each home is registered with its own sectional title under the Sectional Properties Act, 2020. It gives you your home and a share in the land and the common areas, run by a management company the owners control."],
  ["Where does my money go before completion?", "Deposits are paid into an escrow account with the project’s bank and released only as the building reaches each stage, certified by the project’s quantity surveyor."],
  ["What if the building is late?", "The agreement for sale sets the completion date and what you are owed if it slips. You can follow every stage on the progress page."],
  ["Can I choose my finishes?", "Yes, from two kitchen colourways and two floor tones, until the fit-out freeze about a year before completion."],
  ["Is parking included?", "Yes: one space with every one- and two-bedroom home and two with three-bedroom homes and penthouses. Extra spaces can be bought while they last."],
  ["What does the service charge cover?", "Security, cleaning, the gardens and the river walk, the pool and gym, lifts, the backup generator, insurance of the building, and a fund for repairs in years to come."],
  ["Can I let my home?", "Yes. Long lets need nothing more than notice to the management company. Short lets are allowed with its approval, under the building’s house rules."],
];

/** The price the buying page shows each plan on. */
export const EXAMPLE_PRICE = 18_000_000;

export interface PlanRow {
  shareBP: number;
  label: string;
  amount: number;
  /** Set on a run of instalments: the amount of each. */
  each: number;
}

const numberWords = ["No", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve"];

/** A plan card's lines: each payment, with the construction instalments folded into one. */
export function planRows(s: Schedule): PlanRow[] {
  const out: PlanRow[] = [];
  const total = s.instalments.reduce((sum, i) => sum + i.amount, 0);
  const runLength = s.instalments.filter((i) => i.label.startsWith("Construction instalment")).length;
  let run: PlanRow | null = null;
  for (const i of s.instalments) {
    if (i.label.startsWith("Construction instalment")) {
      if (!run) {
        run = { shareBP: 0, label: "", amount: 0, each: i.amount };
        out.push(run);
      }
      run.amount += i.amount;
      // The run's share from its money, so seven payments of 5.71% read as
      // the 40% they make, not 39.97%.
      if (total > 0) run.shareBP = Math.floor((run.amount * 10000 + Math.floor(total / 2)) / total);
      run.label = `${numberWords[runLength] ?? String(runLength)} quarterly instalments while it rises`;
      continue;
    }
    out.push({ shareBP: i.percentBP, label: i.label, amount: i.amount, each: 0 });
  }
  return out;
}
