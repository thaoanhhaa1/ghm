// Renders the ghm app icon: a liquid-glass squircle with a git-branch glyph.
// Usage: swift app/scripts/make-icon.swift <out.iconset>
import AppKit
import CoreGraphics
import Foundation

let outDir = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "AppIcon.iconset"
let size: CGFloat = 1024

func color(_ r: CGFloat, _ g: CGFloat, _ b: CGFloat, _ a: CGFloat = 1) -> CGColor {
    CGColor(srgbRed: r / 255, green: g / 255, blue: b / 255, alpha: a)
}

let space = CGColorSpace(name: CGColorSpace.sRGB)!
guard let ctx = CGContext(
    data: nil, width: Int(size), height: Int(size), bitsPerComponent: 8,
    bytesPerRow: 0, space: space,
    bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
) else { fatalError("context") }

ctx.setAllowsAntialiasing(true)
ctx.interpolationQuality = .high

// MARK: squircle

let margin: CGFloat = 56
let rect = CGRect(x: margin, y: margin, width: size - margin * 2, height: size - margin * 2)
let radius: CGFloat = 220
let squircle = CGPath(roundedRect: rect, cornerWidth: radius, cornerHeight: radius, transform: nil)

ctx.saveGState()
ctx.addPath(squircle)
ctx.clip()

// Deep indigo → violet → teal diagonal base.
let base = CGGradient(colorsSpace: space, colors: [
    color(32, 38, 92), color(61, 48, 138), color(20, 96, 120),
] as CFArray, locations: [0, 0.52, 1])!
ctx.drawLinearGradient(base, start: CGPoint(x: rect.minX, y: rect.maxY), end: CGPoint(x: rect.maxX, y: rect.minY), options: [])

// Aurora blobs give the glass depth.
func blob(_ center: CGPoint, _ r: CGFloat, _ c: CGColor, _ alpha: CGFloat) {
    let g = CGGradient(colorsSpace: space, colors: [
        c.copy(alpha: alpha)!, c.copy(alpha: 0)!,
    ] as CFArray, locations: [0, 1])!
    ctx.drawRadialGradient(g, startCenter: center, startRadius: 0, endCenter: center, endRadius: r, options: [])
}
blob(CGPoint(x: 300, y: 820), 430, color(86, 204, 255), 0.55)
blob(CGPoint(x: 790, y: 700), 400, color(170, 120, 255), 0.45)
blob(CGPoint(x: 560, y: 250), 420, color(64, 240, 190), 0.30)

// Soft specular sweep across the top.
let spec = CGGradient(colorsSpace: space, colors: [
    color(255, 255, 255, 0.42), color(255, 255, 255, 0.06), color(255, 255, 255, 0),
] as CFArray, locations: [0, 0.45, 1])!
ctx.drawLinearGradient(spec, start: CGPoint(x: rect.minX, y: rect.maxY), end: CGPoint(x: rect.maxX, y: rect.midY + 120), options: [])

// Inner vignette keeps the edges grounded.
let vignette = CGGradient(colorsSpace: space, colors: [
    color(0, 0, 0, 0), color(0, 0, 0, 0.28),
] as CFArray, locations: [0.55, 1])!
ctx.drawRadialGradient(vignette, startCenter: CGPoint(x: size / 2, y: size / 2), startRadius: 180,
                       endCenter: CGPoint(x: size / 2, y: size / 2), endRadius: 620, options: [])
ctx.restoreGState()

// MARK: glyph

// A git branch: trunk with three commits and one merge branch, drawn as glass.
let trunkX: CGFloat = 404
let glyphBottom: CGFloat = 268
let glyphTop: CGFloat = 760

func node(_ p: CGPoint, _ r: CGFloat, tint: CGColor) {
    ctx.saveGState()
    // Glass body.
    let body = CGGradient(colorsSpace: space, colors: [
        tint.copy(alpha: 0.95)!, color(255, 255, 255, 0.92),
    ] as CFArray, locations: [0, 1])!
    ctx.addEllipse(in: CGRect(x: p.x - r, y: p.y - r, width: r * 2, height: r * 2))
    ctx.clip()
    ctx.drawLinearGradient(body, start: CGPoint(x: p.x, y: p.y + r), end: CGPoint(x: p.x, y: p.y - r), options: [])
    let shine = CGGradient(colorsSpace: space, colors: [
        color(255, 255, 255, 0.9), color(255, 255, 255, 0),
    ] as CFArray, locations: [0, 1])!
    ctx.drawRadialGradient(shine, startCenter: CGPoint(x: p.x - r * 0.35, y: p.y + r * 0.45), startRadius: 0,
                           endCenter: CGPoint(x: p.x - r * 0.35, y: p.y + r * 0.45), endRadius: r * 0.9, options: [])
    ctx.restoreGState()

    // Rim.
    ctx.setStrokeColor(color(255, 255, 255, 0.55))
    ctx.setLineWidth(5)
    ctx.strokeEllipse(in: CGRect(x: p.x - r, y: p.y - r, width: r * 2, height: r * 2))
}

func bar(from a: CGPoint, to b: CGPoint, width: CGFloat) {
    ctx.setLineCap(.round)
    ctx.setStrokeColor(color(255, 255, 255, 0.85))
    ctx.setLineWidth(width)
    ctx.move(to: a)
    ctx.addLine(to: b)
    ctx.strokePath()
    // Inner glow line.
    ctx.setStrokeColor(color(190, 240, 255, 0.5))
    ctx.setLineWidth(width * 0.45)
    ctx.move(to: a)
    ctx.addLine(to: b)
    ctx.strokePath()
}

let top = CGPoint(x: trunkX, y: glyphTop)
let mid = CGPoint(x: trunkX, y: 514)
let bottom = CGPoint(x: trunkX, y: glyphBottom)
let branch = CGPoint(x: 654, y: 640)

bar(from: top, to: bottom, width: 34)

// Merge branch curve from mid to the branch node.
ctx.setLineCap(.round)
ctx.setStrokeColor(color(255, 255, 255, 0.82))
ctx.setLineWidth(30)
ctx.move(to: CGPoint(x: trunkX, y: 560))
ctx.addCurve(to: CGPoint(x: branch.x - 10, y: branch.y - 70),
             control1: CGPoint(x: trunkX + 4, y: 640),
             control2: CGPoint(x: branch.x - 70, y: 640))
ctx.strokePath()

node(top, 62, tint: color(120, 225, 255))
node(mid, 46, tint: color(150, 160, 255))
node(bottom, 62, tint: color(110, 245, 205))
node(branch, 56, tint: color(190, 140, 255))

// Small terminal caret in the lower-right, a nod to the CLI origin.
ctx.setLineCap(.round)
ctx.setStrokeColor(color(255, 255, 255, 0.75))
ctx.setLineWidth(22)
ctx.move(to: CGPoint(x: 640, y: 300))
ctx.addLine(to: CGPoint(x: 700, y: 360))
ctx.addLine(to: CGPoint(x: 640, y: 420))
ctx.strokePath()

// MARK: export

let fm = FileManager.default
try? fm.createDirectory(atPath: outDir, withIntermediateDirectories: true)
let source = CGImageDestinationCreateWithURL(
    URL(fileURLWithPath: outDir + "/icon_512x512@2x.png") as CFURL, "public.png" as CFString, 1, nil
)!
CGImageDestinationAddImage(source, ctx.makeImage()!, nil)
CGImageDestinationFinalize(source)

let generated = outDir + "/icon_512x512@2x.png"
for base in [16, 32, 128, 256, 512] {
    for scale in [1, 2] {
        let px = base * scale
        let dest = "\(outDir)/icon_\(base)x\(base)\(scale == 2 ? "@2x" : "").png"
        if px == 1024 { continue }
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/sips")
        p.arguments = ["-z", "\(px)", "\(px)", generated, "--out", dest]
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        try! p.run()
        p.waitUntilExit()
    }
}
print("wrote \(outDir) from \(generated)")
