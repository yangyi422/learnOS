// Generates the 256×256 plugin logo with no image dependency: pixels are drawn
// at 1024px and box-downsampled 4× (anti-aliasing), then written as a PNG.
//
// Run: npm run logo
import { deflateSync } from 'node:zlib'
import { writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const SIZE = 256
const SCALE = 4
const CANVAS = SIZE * SCALE

const TOP = [0x6a, 0x76, 0xff]
const BOTTOM = [0x4a, 0x5c, 0xf5]

function inRoundedRect(x, y, x0, y0, x1, y1, radius) {
  if (x < x0 || x > x1 || y < y0 || y > y1) return false
  const cx = Math.min(Math.max(x, x0 + radius), x1 - radius)
  const cy = Math.min(Math.max(y, y0 + radius), y1 - radius)
  const dx = x - cx
  const dy = y - cy
  return dx * dx + dy * dy <= radius * radius
}

function inTriangle(px, py, ax, ay, bx, by, cx, cy) {
  const d1 = (px - bx) * (ay - by) - (ax - bx) * (py - by)
  const d2 = (px - cx) * (by - cy) - (bx - cx) * (py - cy)
  const d3 = (px - ax) * (cy - ay) - (cx - ax) * (py - ay)
  const hasNeg = d1 < 0 || d2 < 0 || d3 < 0
  const hasPos = d1 > 0 || d2 > 0 || d3 > 0
  return !(hasNeg && hasPos)
}

const pixels = new Float64Array(CANVAS * CANVAS * 4) // premultiplied RGBA 0..1

function blend(index, r, g, b, alpha) {
  if (alpha <= 0) return
  const a = Math.min(1, alpha)
  for (let c = 0; c < 3; c += 1) {
    const src = [r, g, b][c] / 255
    pixels[index + c] = src * a + pixels[index + c] * (1 - a)
  }
  pixels[index + 3] = a + pixels[index + 3] * (1 - a)
}

for (let y = 0; y < CANVAS; y += 1) {
  for (let x = 0; x < CANVAS; x += 1) {
    const index = (y * CANVAS + x) * 4
    if (!inRoundedRect(x + 0.5, y + 0.5, 0, 0, CANVAS, CANVAS, 224)) continue

    // Vertical brand gradient.
    const t = y / CANVAS
    const r = TOP[0] + (BOTTOM[0] - TOP[0]) * t
    const g = TOP[1] + (BOTTOM[1] - TOP[1]) * t
    const b = TOP[2] + (BOTTOM[2] - TOP[2]) * t
    blend(index, r, g, b, 1)

    // Faint light from the top-left keeps the square from looking flat.
    const dx = (x - CANVAS * 0.24) / CANVAS
    const dy = (y - CANVAS * 0.16) / CANVAS
    const glow = Math.max(0, 1 - Math.sqrt(dx * dx + dy * dy) * 2.1) * 0.14
    blend(index, 255, 255, 255, glow)

    // Shelf: a wide slot the note drops into.
    if (inRoundedRect(x + 0.5, y + 0.5, 244, 748, 780, 818, 35)) {
      blend(index, 255, 255, 255, 0.94)
    }

    // Down arrow: stem + head.
    if (inRoundedRect(x + 0.5, y + 0.5, 462, 250, 562, 566, 46)) {
      blend(index, 255, 255, 255, 1)
    }
    if (inTriangle(x + 0.5, y + 0.5, 372, 552, 652, 552, 512, 706)) {
      blend(index, 255, 255, 255, 1)
    }
  }
}

// Box downsample 4×4, keeping alpha correct.
const out = Buffer.alloc(SIZE * SIZE * 4)
const factor = SCALE * SCALE
for (let y = 0; y < SIZE; y += 1) {
  for (let x = 0; x < SIZE; x += 1) {
    let r = 0
    let g = 0
    let b = 0
    let a = 0
    for (let sy = 0; sy < SCALE; sy += 1) {
      for (let sx = 0; sx < SCALE; sx += 1) {
        const index = ((y * SCALE + sy) * CANVAS + (x * SCALE + sx)) * 4
        r += pixels[index]
        g += pixels[index + 1]
        b += pixels[index + 2]
        a += pixels[index + 3]
      }
    }
    r /= factor
    g /= factor
    b /= factor
    a /= factor
    const offset = (y * SIZE + x) * 4
    if (a <= 0) {
      out[offset] = 0
      out[offset + 1] = 0
      out[offset + 2] = 0
      out[offset + 3] = 0
      continue
    }
    out[offset] = Math.round(Math.min(1, r / a) * 255)
    out[offset + 1] = Math.round(Math.min(1, g / a) * 255)
    out[offset + 2] = Math.round(Math.min(1, b / a) * 255)
    out[offset + 3] = Math.round(a * 255)
  }
}

const CRC_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n += 1) {
    let c = n
    for (let k = 0; k < 8; k += 1) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    }
    table[n] = c >>> 0
  }
  return table
})()

function crc32(buffer) {
  let crc = 0xffffffff
  for (const byte of buffer) {
    crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8)
  }
  return (crc ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
  const length = Buffer.alloc(4)
  length.writeUInt32BE(data.length, 0)
  const body = Buffer.concat([Buffer.from(type, 'ascii'), data])
  const crc = Buffer.alloc(4)
  crc.writeUInt32BE(crc32(body), 0)
  return Buffer.concat([length, body, crc])
}

const ihdr = Buffer.alloc(13)
ihdr.writeUInt32BE(SIZE, 0)
ihdr.writeUInt32BE(SIZE, 4)
ihdr[8] = 8 // bit depth
ihdr[9] = 6 // RGBA
ihdr[10] = 0
ihdr[11] = 0
ihdr[12] = 0

const raw = Buffer.alloc(SIZE * (SIZE * 4 + 1))
for (let y = 0; y < SIZE; y += 1) {
  const rowStart = y * (SIZE * 4 + 1)
  raw[rowStart] = 0 // filter: none
  out.copy(raw, rowStart + 1, y * SIZE * 4, (y + 1) * SIZE * 4)
}

const png = Buffer.concat([
  Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
  chunk('IHDR', ihdr),
  chunk('IDAT', deflateSync(raw, { level: 9 })),
  chunk('IEND', Buffer.alloc(0)),
])

const here = dirname(fileURLToPath(import.meta.url))
const target = resolve(here, '..', 'logo.png')
writeFileSync(target, png)
console.log(`logo written: ${target} (${png.length} bytes, ${SIZE}x${SIZE})`)
