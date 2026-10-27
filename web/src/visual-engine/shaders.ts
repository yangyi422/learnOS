import type { FrameContext, RGB, ShaderScene, SurfaceLight, UniformMap } from './types'

const vertexShader = `#version 300 es
in vec2 aPosition;
void main() {
  gl_Position = vec4(aPosition, 0.0, 1.0);
}`

/**
 * Scheme A — Spatial Ambient.
 *
 * Three light sources outside the viewport, matching the CSS HomeLightField
 * parameters in design.md §3. Over CSS it adds only a very slow drift and a
 * ±6% brightness breath. The falloff is linear because radial-gradient alpha
 * interpolates linearly; smoothstep made the light noticeably weaker.
 */
/**
 * Glass composite pass (admin-only refraction).
 *
 * The ambient scene is rendered into an offscreen texture first; this pass
 * draws that texture back unchanged and, inside the focus card's rounded rect,
 * samples it with a small edge-weighted UV displacement plus a Fresnel rim.
 * Outside the card the pixels are identical to the plain ambient frame.
 */
export const glassVertexShader = vertexShader

export const glassFragmentShader = `#version 300 es
precision highp float;

uniform sampler2D uBg;
uniform vec2 uResolution;
uniform vec4 uRect;
uniform float uRadius;
uniform float uStrength;
uniform float uTheme;

out vec4 outColor;

float sdRoundRect(vec2 p, vec2 halfSize, float radius) {
  vec2 q = abs(p) - halfSize + radius;
  return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - radius;
}

void main() {
  vec2 uv = gl_FragCoord.xy / uResolution;
  vec2 p = gl_FragCoord.xy - uRect.xy - uRect.zw * 0.5;
  float d = sdRoundRect(p, uRect.zw * 0.5, uRadius);

  // Outside the card: reproduce the ambient pixel exactly (no seam).
  if (d > 0.0) {
    outColor = texture(uBg, uv);
    return;
  }

  // Screen-space derivative of the SDF: a continuous edge normal, no stair steps.
  vec2 normal = normalize(vec2(dFdx(d), dFdy(d)) + vec2(1e-5));

  // Refraction lives in a narrow band inside the border; the centre stays still.
  float rim = 1.0 - smoothstep(0.0, 30.0, -d);
  vec2 offset = normal * rim * rim * uStrength / uResolution;
  vec3 base = texture(uBg, uv + offset).rgb;

  vec3 tint = mix(vec3(0.42, 0.52, 0.60), vec3(0.72, 0.66, 0.92), uTheme);
  // Tint only the rim, and keep it low so the corner does not read as plastic.
  vec3 color = base + tint * pow(rim, 2.0) * 0.055;
  outColor = vec4(color, 1.0);
}`

const ambientFragment = `#version 300 es
precision highp float;

uniform vec2 uResolution;
uniform float uTime;
uniform float uStrength;
uniform vec3 uGroundA;
uniform vec3 uGroundB;
uniform vec3 uGroundC;
uniform float uGroundGradient;
uniform float uGroundVertical;
uniform vec4 uEntryField;
uniform vec4 uReflectionField;
uniform vec4 uFloorField;
uniform vec3 uEntry;
uniform float uEntryAlpha;
uniform vec3 uReflection;
uniform float uReflectionAlpha;
uniform vec3 uFloor;
uniform float uFloorAlpha;

out vec4 outColor;

const float TAU = 6.28318530718;

float interleavedGradientNoise(vec2 point) {
  return fract(52.9829189 * fract(dot(point, vec2(0.06711056, 0.00583715))));
}

float radial(vec2 point, vec2 center, vec2 radius, float stopAt) {
  vec2 delta = (point - center) / radius;
  return clamp(1.0 - length(delta) / stopAt, 0.0, 1.0);
}

void main() {
  vec2 uv = gl_FragCoord.xy / uResolution;
  vec2 point = vec2(uv.x, 1.0 - uv.y);
  float time = uTime;

  vec3 ground = uGroundB;
  if (uGroundGradient > 0.5) {
    float axis = uGroundVertical > 0.5 ? point.y
      : clamp(dot(point - 0.5, vec2(0.70710678, 0.70710678)) + 0.5, 0.0, 1.0);
    ground = mix(uGroundA, uGroundB, smoothstep(0.0, 0.55, axis));
    ground = mix(ground, uGroundC, smoothstep(0.55, 1.0, axis));
  }

  vec2 entryCenter = uEntryField.xy + vec2(0.015 * sin(time * TAU / 37.0), 0.012 * sin(time * TAU / 61.0 + 1.3));
  vec2 entryRadius = uEntryField.zw * vec2(1.0 + 0.02 * sin(time * TAU / 53.0), 1.0 + 0.02 * sin(time * TAU / 47.0 + 2.1));
  vec2 reflectionCenter = uReflectionField.xy + vec2(0.012 * sin(time * TAU / 43.0 + 0.7), 0.010 * sin(time * TAU / 67.0 + 2.4));
  vec2 reflectionRadius = uReflectionField.zw * vec2(1.0 + 0.015 * sin(time * TAU / 59.0 + 1.0), 1.0 + 0.015 * sin(time * TAU / 71.0));
  vec2 floorCenter = uFloorField.xy + vec2(0.010 * sin(time * TAU / 73.0), 0.006 * sin(time * TAU / 41.0 + 1.8));
  vec2 floorRadius = uFloorField.zw * vec2(1.0, 1.0 + 0.02 * sin(time * TAU / 37.0));

  float entry = radial(point, entryCenter, entryRadius, 0.82) * uEntryAlpha;
  float reflection = radial(point, reflectionCenter, reflectionRadius, 0.85) * uReflectionAlpha;
  float floorLight = radial(point, floorCenter, floorRadius, 0.88) * uFloorAlpha;
  float breath = 1.0 + 0.06 * sin(time * TAU / 53.0);

  vec3 color = ground;
  color = mix(color, uEntry, entry * uStrength * breath);
  color = mix(color, uReflection, reflection * uStrength * breath);
  color = mix(color, uFloor, floorLight * uStrength * breath);

  // Dither keeps the large gradients free of 8-bit banding.
  color += (interleavedGradientNoise(gl_FragCoord.xy) - 0.5) * (1.5 / 255.0);
  outColor = vec4(color, 1.0);
}`

/**
 * Scheme B — Material Depth, an experiment confined to the lab stage.
 *
 * 3-octave value noise height field, differential normals, one raking light
 * band. Grain is static: animated grain reads as shimmer behind text.
 */
const materialFragment = `#version 300 es
precision highp float;

uniform vec2 uResolution;
uniform float uTime;
uniform float uStrength;
uniform vec3 uBase;
uniform vec3 uShadow;
uniform vec3 uHighlight;
uniform float uNormalStrength;
uniform float uSpecularPower;

out vec4 outColor;

const float TAU = 6.28318530718;

float hash(vec2 point) {
  point = fract(point * vec2(123.34, 456.21));
  point += dot(point, point + 45.32);
  return fract(point.x * point.y);
}

float noise(vec2 point) {
  vec2 cell = floor(point);
  vec2 local = fract(point);
  local = local * local * (3.0 - 2.0 * local);
  float a = hash(cell);
  float b = hash(cell + vec2(1.0, 0.0));
  float c = hash(cell + vec2(0.0, 1.0));
  float d = hash(cell + vec2(1.0, 1.0));
  return mix(mix(a, b, local.x), mix(c, d, local.x), local.y);
}

float fbm(vec2 point) {
  float sum = 0.0;
  float amplitude = 0.5;
  for (int i = 0; i < 3; i++) {
    sum += amplitude * noise(point);
    point *= 2.03;
    amplitude *= 0.5;
  }
  return sum;
}

float height(vec2 point, float aspect) {
  return fbm(point * vec2(aspect, 1.0) * 14.0);
}

void main() {
  vec2 uv = gl_FragCoord.xy / uResolution;
  vec2 point = vec2(uv.x, 1.0 - uv.y);
  float aspect = uResolution.x / uResolution.y;
  vec2 epsilon = vec2(2.0) / uResolution;

  float dx = height(point + vec2(epsilon.x, 0.0), aspect) - height(point - vec2(epsilon.x, 0.0), aspect);
  float dy = height(point + vec2(0.0, epsilon.y), aspect) - height(point - vec2(0.0, epsilon.y), aspect);
  vec3 normal = normalize(vec3(-dx * uNormalStrength, -dy * uNormalStrength, 1.0));

  float angle = TAU / 3.0 * (uTime / 48.0);
  vec3 lightDirection = normalize(vec3(cos(angle), sin(angle) * 0.45, 0.55));
  vec3 halfway = normalize(lightDirection + vec3(0.0, 0.0, 1.0));
  float diffuse = max(dot(normal, lightDirection), 0.0);
  float specular = pow(max(dot(normal, halfway), 0.0), uSpecularPower);

  vec3 color = mix(uShadow, uBase, smoothstep(-0.35, 0.75, diffuse));
  color += uHighlight * specular * 0.45 * uStrength;
  color += uHighlight * pow(specular, 3.0) * 0.10 * uStrength;
  color += (hash(gl_FragCoord.xy) - 0.5) * (1.5 / 255.0);
  outColor = vec4(color, 1.0);
}`

const ambientUniforms = [
  'uResolution', 'uTime', 'uStrength',
  'uGroundA', 'uGroundB', 'uGroundC', 'uGroundGradient', 'uGroundVertical',
  'uEntryField', 'uReflectionField', 'uFloorField',
  'uEntry', 'uEntryAlpha', 'uReflection', 'uReflectionAlpha', 'uFloor', 'uFloorAlpha',
] as const

const materialUniforms = [
  'uResolution', 'uTime', 'uStrength',
  'uBase', 'uShadow', 'uHighlight', 'uNormalStrength', 'uSpecularPower',
] as const

function writeColor(gl: WebGL2RenderingContext, location: WebGLUniformLocation | null, color: RGB): void {
  gl.uniform3f(location, color[0], color[1], color[2])
}

function writeAmbient(gl: WebGL2RenderingContext, uniforms: UniformMap, frame: FrameContext): void {
  const { light } = frame
  const ground = light.ground.length ? light.ground : ([[0, 0, 0]] as RGB[])
  const first = ground[0]!
  const second = ground[1] ?? first
  const third = ground[2] ?? second
  writeColor(gl, uniforms.uGroundA ?? null, first)
  writeColor(gl, uniforms.uGroundB ?? null, second)
  writeColor(gl, uniforms.uGroundC ?? null, third)
  gl.uniform1f(uniforms.uGroundGradient ?? null, light.gradient ? 1 : 0)
  gl.uniform1f(uniforms.uGroundVertical ?? null, light.vertical ? 1 : 0)
  gl.uniform4f(uniforms.uEntryField ?? null, ...(light.fields?.entry ?? [-0.12, 0.24, 0.76, 0.48]))
  gl.uniform4f(uniforms.uReflectionField ?? null, ...(light.fields?.reflection ?? [1.12, 0.66, 0.65, 0.52]))
  gl.uniform4f(uniforms.uFloorField ?? null, ...(light.fields?.floor ?? [0.12, 1.15, 0.70, 0.30]))

  writeColor(gl, uniforms.uEntry ?? null, light.entry.color)
  gl.uniform1f(uniforms.uEntryAlpha ?? null, light.entry.alpha)
  writeColor(gl, uniforms.uReflection ?? null, light.reflection.color)
  gl.uniform1f(uniforms.uReflectionAlpha ?? null, light.reflection.alpha)
  writeColor(gl, uniforms.uFloor ?? null, light.floor.color)
  gl.uniform1f(uniforms.uFloorAlpha ?? null, light.floor.alpha)
}

/** Material palette is B's own; only the light direction follows the theme. */
const MATERIAL_LIGHT = { base: [0.890, 0.929, 0.965], shadow: [0.847, 0.890, 0.941], highlight: [0.980, 0.992, 1.0], normal: 5.0 }
const MATERIAL_DARK = { base: [0.063, 0.047, 0.133], shadow: [0.027, 0.020, 0.063], highlight: [0.550, 0.420, 0.850], normal: 7.0 }

function writeMaterial(gl: WebGL2RenderingContext, uniforms: UniformMap, frame: FrameContext): void {
  const palette = frame.light.theme === 'dark' ? MATERIAL_DARK : MATERIAL_LIGHT
  writeColor(gl, uniforms.uBase ?? null, palette.base as RGB)
  writeColor(gl, uniforms.uShadow ?? null, palette.shadow as RGB)
  writeColor(gl, uniforms.uHighlight ?? null, palette.highlight as RGB)
  gl.uniform1f(uniforms.uNormalStrength ?? null, palette.normal)
  gl.uniform1f(uniforms.uSpecularPower ?? null, 24)
}

function frameUniforms(gl: WebGL2RenderingContext, uniforms: UniformMap, frame: FrameContext): void {
  gl.uniform2f(uniforms.uResolution ?? null, frame.width, frame.height)
  gl.uniform1f(uniforms.uTime ?? null, frame.time)
  gl.uniform1f(uniforms.uStrength ?? null, frame.strength)
}

export const ambientScene: ShaderScene = {
  id: 'ambient',
  label: 'A · Spatial Ambient',
  vertexShader,
  fragmentShader: ambientFragment,
  uniformNames: ambientUniforms,
  write(gl, uniforms, frame) {
    frameUniforms(gl, uniforms, frame)
    writeAmbient(gl, uniforms, frame)
  },
}

export const materialScene: ShaderScene = {
  id: 'material',
  label: 'B · Material Depth',
  vertexShader,
  fragmentShader: materialFragment,
  uniformNames: materialUniforms,
  write(gl, uniforms, frame) {
    frameUniforms(gl, uniforms, frame)
    writeMaterial(gl, uniforms, frame)
  },
}

export const scenes: Record<ShaderScene['id'], ShaderScene> = {
  ambient: ambientScene,
  material: materialScene,
}

export type { SurfaceLight }
