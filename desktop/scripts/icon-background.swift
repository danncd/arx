import AppKit

let source = NSImage(contentsOfFile: CommandLine.arguments[1])!
let destination = CommandLine.arguments[2]
let size = 1024
let bitmap = NSBitmapImageRep(
    bitmapDataPlanes: nil,
    pixelsWide: size,
    pixelsHigh: size,
    bitsPerSample: 8,
    samplesPerPixel: 4,
    hasAlpha: true,
    isPlanar: false,
    colorSpaceName: .deviceRGB,
    bytesPerRow: 0,
    bitsPerPixel: 0
)!

NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: bitmap)
NSColor.white.setFill()
NSRect(x: 0, y: 0, width: size, height: size).fill()
source.draw(
    in: NSRect(x: 41, y: 41, width: size - 82, height: size - 82),
    from: .zero,
    operation: .sourceOver,
    fraction: 1
)
NSGraphicsContext.current?.flushGraphics()
NSGraphicsContext.restoreGraphicsState()

try bitmap.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: destination))
