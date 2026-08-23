import { customAlphabet } from 'nanoid'

/** Short readable segments; combined as two-word session titles. */
const WORDS = [
  'amber',
  'bold',
  'calm',
  'cedar',
  'coral',
  'crisp',
  'delta',
  'ember',
  'fern',
  'frost',
  'glow',
  'haven',
  'ivory',
  'jade',
  'keen',
  'lake',
  'lotus',
  'maple',
  'mist',
  'north',
  'olive',
  'orbit',
  'pearl',
  'pine',
  'plume',
  'quartz',
  'river',
  'sage',
  'silver',
  'soft',
  'stone',
  'swift',
  'tide',
  'umber',
  'vale',
  'wave',
  'willow',
  'zephyr',
] as const

const pickIndex = customAlphabet('0123456789', 6)

function pickWord(): string {
  const n = Number.parseInt(pickIndex(), 10)
  return WORDS[n % WORDS.length]!
}

/** Random two-word session title (nanoid-driven picks), e.g. "calm river". */
export function randomSessionTitle(): string {
  let a = pickWord()
  let b = pickWord()
  // Avoid identical pair when the list allows a second distinct pick.
  if (a === b && WORDS.length > 1) {
    b = WORDS[(WORDS.indexOf(a as (typeof WORDS)[number]) + 1 + (Number.parseInt(pickIndex(), 10) % (WORDS.length - 1))) % WORDS.length]!
  }
  return `${a} ${b}`
}
