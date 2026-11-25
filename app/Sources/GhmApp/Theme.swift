import SwiftUI

extension Color {
    init(hex: UInt32, alpha: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }

    // iOS system colours. The wallpaper is the only place they are used at full
    // strength; on glass they act as tints, which is how iOS 26 reads.
    static let iosBlue = Color(hex: 0x0A84FF)
    static let iosIndigo = Color(hex: 0x5E5CE6)
    static let iosPurple = Color(hex: 0xBF5AF2)
    static let iosPink = Color(hex: 0xFF375F)
    static let iosTeal = Color(hex: 0x40C8E0)
    static let iosGreen = Color(hex: 0x30D158)
    static let iosOrange = Color(hex: 0xFF9F0A)

    static let ghmAccent = Color.iosBlue
    static let ghmPositive = Color.iosGreen
    static let ghmWarning = Color.iosOrange
    static let ghmInfo = Color.iosTeal
    static let ghmDanger = Color.iosPink
}

/// iPhone 26 wallpaper: soft colour blooms on deep indigo. Everything above it
/// is glass, so the colour is what makes the material read as glass instead of
/// grey frosted plastic.
struct LiquidBackground: View {
    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            ZStack {
                LinearGradient(
                    colors: [Color(hex: 0x05060F), Color(hex: 0x0B0A20), Color(hex: 0x140A24)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )

                bloom(Color.iosBlue.opacity(0.42), w * 0.95, blur: 170)
                    .position(x: w * 0.14, y: h * 0.08)

                bloom(Color.iosIndigo.opacity(0.38), w * 0.8, blur: 180)
                    .position(x: w * 0.94, y: h * 0.12)

                bloom(Color.iosPurple.opacity(0.16), w * 0.7, blur: 200)
                    .position(x: w * 0.74, y: h * 1.04)

                bloom(Color.iosTeal.opacity(0.16), w * 0.6, blur: 200)
                    .position(x: w * 0.06, y: h * 0.92)

                // Dark-mode wallpaper dimming: keeps the colour visible at the
                // edges while glass panels and white text stay readable.
                Color.black.opacity(0.5)

                RadialGradient(
                    colors: [.clear, .black.opacity(0.34)],
                    center: .center,
                    startRadius: 220,
                    endRadius: 1100
                )
                .ignoresSafeArea()
            }
            .ignoresSafeArea()
        }
    }

    private func bloom(_ color: Color, _ size: CGFloat, blur: CGFloat) -> some View {
        Circle()
            .fill(color)
            .frame(width: size, height: size)
            .blur(radius: blur)
    }
}

extension View {
    /// A floating Liquid Glass island.
    func glassPanel(_ radius: CGFloat = 28, tint: Color? = nil, interactive: Bool = false) -> some View {
        modifier(GlassPanelModifier(radius: radius, tint: tint, interactive: interactive))
    }
}

private struct GlassPanelModifier: ViewModifier {
    let radius: CGFloat
    let tint: Color?
    let interactive: Bool

    func body(content: Content) -> some View {
        content
            .glassEffect(glass, in: .rect(cornerRadius: radius))
            .overlay(
                RoundedRectangle(cornerRadius: radius)
                    .strokeBorder(
                        LinearGradient(
                            colors: [.white.opacity(0.32), .white.opacity(0.06), .white.opacity(0.02)],
                            startPoint: .top,
                            endPoint: .bottom
                        ),
                        lineWidth: 0.8
                    )
            )
    }

    private var glass: Glass {
        // A dim scrim keeps the islands readable over a colourful wallpaper;
        // without it the glass just transmits the blue and magenta blooms.
        var glass: Glass = .regular
        if let tint {
            glass = glass.tint(tint)
        }
        if interactive {
            glass = glass.interactive()
        }
        return glass
    }
}

/// Capsule counter, the iOS 26 "glass chip".
struct GlassBadge: View {
    let text: String
    var systemImage: String?
    var tint: Color = .secondary

    var body: some View {
        HStack(spacing: 5) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.caption2.weight(.semibold))
            }
            Text(text)
                .font(.caption.weight(.medium).monospacedDigit())
        }
        .foregroundStyle(tint)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(.white.opacity(0.10), in: .capsule)
        .overlay(Capsule().strokeBorder(.white.opacity(0.10), lineWidth: 0.5))
    }
}

/// Circular icon button, like the trailing items of an iOS navigation bar.
struct GlassIconButton: View {
    let systemImage: String
    let help: String
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: 13, weight: .semibold))
                .frame(width: 22, height: 22)
        }
        .buttonStyle(.glass)
        .buttonBorderShape(.circle)
        .help(help)
    }
}

struct StatePill: View {
    let repo: Repo

    var body: some View {
        Text(repo.stateLabel)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 9)
            .padding(.vertical, 3)
            .foregroundStyle(foreground)
            .background(background, in: .capsule)
            .overlay(
                Capsule().strokeBorder(.white.opacity(0.12), lineWidth: 0.5)
            )
    }

    private var foreground: Color {
        if repo.dirty > 0 { return .ghmWarning }
        if repo.behind > 0 { return .ghmInfo }
        if repo.ahead > 0 { return .ghmPositive }
        return .secondary
    }

    private var background: Color {
        if repo.dirty > 0 { return .ghmWarning.opacity(0.18) }
        if repo.behind > 0 { return .ghmInfo.opacity(0.18) }
        if repo.ahead > 0 { return .ghmPositive.opacity(0.18) }
        return .white.opacity(0.07)
    }
}

/// iOS contact-style avatar: a vivid gradient disc per login.
struct Avatar: View {
    let account: Account
    var size: CGFloat = 28

    var body: some View {
        ZStack {
            Circle().fill(
                LinearGradient(colors: tint, startPoint: .topLeading, endPoint: .bottomTrailing)
            )
            Text(account.initials)
                .font(.system(size: size * 0.36, weight: .bold))
                .foregroundStyle(.white)
        }
        .frame(width: size, height: size)
        .overlay(Circle().strokeBorder(.white.opacity(0.30), lineWidth: 0.8))
    }

    private var tint: [Color] {
        let palette: [[Color]] = [
            [.iosBlue, .iosIndigo],
            [.iosPurple, .iosPink],
            [.iosTeal, .iosBlue],
            [.iosOrange, .iosPink],
        ]
        let sum = account.login.unicodeScalars.reduce(0) { $0 + Int($1.value) }
        return palette[sum % palette.count]
    }
}

/// Floating bottom bar item: icon over a caption, dimmed until it applies,
/// tinted when active — the iOS 26 tab bar pattern.
struct ActionBarItem: View {
    let systemImage: String
    let title: String
    var tint: Color = .primary
    var enabled: Bool = true
    var action: () -> Void

    @State private var hovering = false

    var body: some View {
        Button(action: action) {
            VStack(spacing: 3) {
                Image(systemName: systemImage)
                    .font(.system(size: 15, weight: .semibold))
                    .frame(height: 18)
                Text(title)
                    .font(.system(size: 10, weight: .medium))
            }
            .foregroundStyle(enabled ? tint : Color.secondary.opacity(0.5))
            .frame(width: 58, height: 42)
            .background(
                hovering && enabled ? AnyShapeStyle(.white.opacity(0.08)) : AnyShapeStyle(.clear),
                in: .rect(cornerRadius: 12)
            )
            .contentShape(.rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
        .disabled(!enabled)
        .onHover { hovering = $0 }
        .help(title)
    }
}

/// Backdrop for sheets: the same wallpaper idea, dimmed so glass cards on top
/// keep their contrast.
struct SheetBackdrop: View {
    var body: some View {
        ZStack {
            LinearGradient(
                colors: [Color(hex: 0x0B0E22), Color(hex: 0x1A1233)],
                startPoint: .top,
                endPoint: .bottom
            )
            Circle().fill(Color.iosBlue.opacity(0.34)).frame(width: 460).blur(radius: 130)
                .offset(x: -180, y: -220)
            Circle().fill(Color.iosPurple.opacity(0.28)).frame(width: 400).blur(radius: 130)
                .offset(x: 200, y: 140)
            Circle().fill(Color.iosPink.opacity(0.18)).frame(width: 320).blur(radius: 130)
                .offset(x: 140, y: -160)
        }
        .ignoresSafeArea()
    }
}
